#include <gtest/gtest.h>

#include <atomic>
#include <thread>
#include <vector>

#include "mwal/status.h"
#include "mwal/write_batch.h"
#include "wal/write_thread.h"

namespace mwal {

TEST(WriteThreadTest, SingleWriter) {
  WriteThread wt;
  WriteThread::Writer w;
  WriteBatch batch;
  batch.Put("k", "v");
  w.batch = &batch;
  w.sync = false;

  bool is_leader = wt.JoinBatchGroup(&w);
  ASSERT_TRUE(is_leader);

  WriteThread::WriteGroup group;
  size_t sz = wt.EnterAsBatchGroupLeader(&w, &group);
  EXPECT_EQ(sz, 1u);
  EXPECT_EQ(group.leader, &w);
  EXPECT_EQ(group.last_writer, &w);
  EXPECT_FALSE(group.need_sync);

  wt.ExitAsBatchGroupLeader(group, Status::OK());
}

TEST(WriteThreadTest, TwoWriters) {
  WriteThread wt;

  std::atomic<int> completed{0};
  std::vector<Status> results(2);
  WriteBatch batch1, batch2;
  batch1.Put("k1", "v1");
  batch2.Put("k2", "v2");

  auto run = [&](int id, WriteBatch* batch) {
    WriteThread::Writer w;
    w.batch = batch;
    bool is_leader = wt.JoinBatchGroup(&w);
    if (is_leader) {
      // Give other thread a chance to enqueue.
      std::this_thread::sleep_for(std::chrono::milliseconds(30));
      WriteThread::WriteGroup group;
      wt.EnterAsBatchGroupLeader(&w, &group);
      wt.ExitAsBatchGroupLeader(group, Status::OK());
      results[id] = Status::OK();
    } else {
      results[id] = w.status;
    }
    completed.fetch_add(1);
  };

  std::thread t1(run, 0, &batch1);
  std::thread t2(run, 1, &batch2);
  t1.join();
  t2.join();

  EXPECT_EQ(completed.load(), 2);
  EXPECT_TRUE(results[0].ok());
  EXPECT_TRUE(results[1].ok());
}

TEST(WriteThreadTest, ManyWriters) {
  WriteThread wt;
  constexpr int kNumThreads = 8;
  std::atomic<int> completed{0};
  std::vector<Status> statuses(kNumThreads);
  std::vector<WriteBatch> batches(kNumThreads);

  for (int i = 0; i < kNumThreads; i++) {
    std::string key = "key_" + std::to_string(i);
    std::string val = "val_" + std::to_string(i);
    batches[i].Put(key, val);
  }

  std::vector<std::thread> threads;
  for (int i = 0; i < kNumThreads; i++) {
    threads.emplace_back([&, i] {
      WriteThread::Writer w;
      w.batch = &batches[i];
      bool is_leader = wt.JoinBatchGroup(&w);
      if (is_leader) {
        WriteThread::WriteGroup group;
        wt.EnterAsBatchGroupLeader(&w, &group);
        wt.ExitAsBatchGroupLeader(group, Status::OK());
        statuses[i] = Status::OK();
      } else {
        statuses[i] = w.status;
      }
      completed.fetch_add(1);
    });
  }

  for (auto& t : threads) t.join();
  EXPECT_EQ(completed.load(), kNumThreads);
  for (int i = 0; i < kNumThreads; i++) {
    EXPECT_TRUE(statuses[i].ok()) << "Thread " << i << ": " << statuses[i].ToString();
  }
}

TEST(WriteThreadTest, LeaderSyncCoalescing) {
  WriteThread wt;

  WriteThread::Writer w1, w2;
  WriteBatch b1, b2;
  b1.Put("k1", "v1");
  b2.Put("k2", "v2");
  w1.batch = &b1;
  w1.sync = true;
  w2.batch = &b2;
  w2.sync = false;

  // w1 is leader.
  bool is_leader = wt.JoinBatchGroup(&w1);
  ASSERT_TRUE(is_leader);

  WriteThread::WriteGroup group;
  wt.EnterAsBatchGroupLeader(&w1, &group);
  // Since w1 has sync=true, group should need_sync even though w2 doesn't.
  EXPECT_TRUE(group.need_sync);

  wt.ExitAsBatchGroupLeader(group, Status::OK());
}

TEST(WriteThreadTest, ErrorPropagation) {
  WriteThread wt;

  Status follower_result;
  WriteBatch b1, b2;
  b1.Put("k1", "v1");
  b2.Put("k2", "v2");

  auto leader_fn = [&] {
    WriteThread::Writer w;
    w.batch = &b1;
    bool is_leader = wt.JoinBatchGroup(&w);
    ASSERT_TRUE(is_leader);
    // Wait for follower to enqueue.
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    WriteThread::WriteGroup group;
    wt.EnterAsBatchGroupLeader(&w, &group);
    EXPECT_GE(group.size, 2u);
    wt.ExitAsBatchGroupLeader(group, Status::IOError("disk full"));
  };

  auto follower_fn = [&] {
    WriteThread::Writer w;
    w.batch = &b2;
    bool is_leader = wt.JoinBatchGroup(&w);
    if (!is_leader) {
      follower_result = w.status;
    } else {
      WriteThread::WriteGroup group;
      wt.EnterAsBatchGroupLeader(&w, &group);
      wt.ExitAsBatchGroupLeader(group, Status::OK());
      follower_result = Status::OK();
    }
  };

  std::thread t1(leader_fn);
  std::this_thread::sleep_for(std::chrono::milliseconds(10));
  std::thread t2(follower_fn);
  t1.join();
  t2.join();

  EXPECT_TRUE(follower_result.IsIOError()) << follower_result.ToString();
}

}  // namespace mwal
