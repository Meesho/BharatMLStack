// Leader-follower write thread for group commit in mwal.

#pragma once

#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <mutex>

#include "mwal/status.h"
#include "mwal/write_batch.h"

namespace mwal {

class WriteThread {
 public:
  struct Writer {
    WriteBatch* batch = nullptr;
    bool sync = false;
    bool disable_wal = false;
    bool done = false;
    bool is_leader = false;
    Status status;
    std::condition_variable cv;

    Writer* link_newer = nullptr;

    Writer() = default;
    Writer(const Writer&) = delete;
    Writer& operator=(const Writer&) = delete;
  };

  struct WriteGroup {
    Writer* leader = nullptr;
    Writer* last_writer = nullptr;
    size_t size = 0;
    bool need_sync = false;
  };

  WriteThread();
  ~WriteThread() = default;

  WriteThread(const WriteThread&) = delete;
  WriteThread& operator=(const WriteThread&) = delete;

  // Enqueue writer. Returns true if this thread is the leader.
  // If follower, blocks until leader marks it done.
  // Returns false with w->done=true and Aborted status when shutting down.
  bool JoinBatchGroup(Writer* w);

  // Combined join + group formation in a single lock acquisition.
  // Returns true if this thread is the leader (group is populated).
  // Returns false if follower (blocks until done, group is unused).
  bool JoinAndBuildGroup(Writer* w, WriteGroup* group,
                         size_t max_group_size = 0);

  // Leader collects all pending writers into a group. Returns group size.
  // max_group_size: 0 = no limit; >0 = cap at this many writers.
  size_t EnterAsBatchGroupLeader(Writer* leader, WriteGroup* group,
                                 size_t max_group_size = 0);

  // Release non-sync followers from the group. Call before fsync to
  // unblock async writers that don't need durability guarantees.
  void CompleteAsyncFollowers(WriteGroup& group, Status status);

  // Leader signals all remaining followers with the final status.
  void ExitAsBatchGroupLeader(WriteGroup& group, Status status);

  // Drain pending writers with Aborted, prevent new joins.
  void DrainAndClose();

 private:
  std::mutex mu_;
  std::deque<Writer*> pending_;
  bool leader_active_ = false;
  bool closed_ = false;
};

}  // namespace mwal
