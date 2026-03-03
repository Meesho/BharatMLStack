#include <gtest/gtest.h>

#include <string>
#include <vector>

#include "mwal/write_batch.h"

namespace mwal {

class WriteBatchTestHandler : public WriteBatch::Handler {
 public:
  struct Entry {
    std::string op;
    std::string key;
    std::string value;
  };
  std::vector<Entry> entries;
  std::vector<std::string> log_data;

  Status Put(const Slice& key, const Slice& value) override {
    entries.push_back({"Put", key.ToString(), value.ToString()});
    return Status::OK();
  }
  Status Delete(const Slice& key) override {
    entries.push_back({"Delete", key.ToString(), ""});
    return Status::OK();
  }
  Status Merge(const Slice& key, const Slice& value) override {
    entries.push_back({"Merge", key.ToString(), value.ToString()});
    return Status::OK();
  }
  void LogData(const Slice& blob) override {
    log_data.push_back(blob.ToString());
  }
};

TEST(WriteBatchTest, Empty) {
  WriteBatch batch;
  EXPECT_EQ(batch.Count(), 0);
  EXPECT_GE(batch.GetDataSize(), 12u);
}

TEST(WriteBatchTest, SinglePut) {
  WriteBatch batch;
  batch.Put("key1", "val1");
  EXPECT_EQ(batch.Count(), 1);
  EXPECT_TRUE(batch.HasPut());

  WriteBatchTestHandler handler;
  ASSERT_TRUE(batch.Iterate(&handler).ok());
  ASSERT_EQ(handler.entries.size(), 1u);
  EXPECT_EQ(handler.entries[0].op, "Put");
  EXPECT_EQ(handler.entries[0].key, "key1");
  EXPECT_EQ(handler.entries[0].value, "val1");
}

TEST(WriteBatchTest, SingleDelete) {
  WriteBatch batch;
  batch.Delete("key1");
  EXPECT_EQ(batch.Count(), 1);

  WriteBatchTestHandler handler;
  ASSERT_TRUE(batch.Iterate(&handler).ok());
  ASSERT_EQ(handler.entries.size(), 1u);
  EXPECT_EQ(handler.entries[0].op, "Delete");
  EXPECT_EQ(handler.entries[0].key, "key1");
}

TEST(WriteBatchTest, MultipleOps) {
  WriteBatch batch;
  batch.Put("k1", "v1");
  batch.Delete("k2");
  batch.Merge("k3", "v3");
  EXPECT_EQ(batch.Count(), 3);

  WriteBatchTestHandler handler;
  ASSERT_TRUE(batch.Iterate(&handler).ok());
  ASSERT_EQ(handler.entries.size(), 3u);
  EXPECT_EQ(handler.entries[0].op, "Put");
  EXPECT_EQ(handler.entries[1].op, "Delete");
  EXPECT_EQ(handler.entries[2].op, "Merge");
}

TEST(WriteBatchTest, SequenceNumber) {
  WriteBatch batch;
  batch.SetSequence(12345);
  EXPECT_EQ(batch.Sequence(), 12345u);
}

TEST(WriteBatchTest, PutLogData) {
  WriteBatch batch;
  batch.Put("key", "val");
  batch.PutLogData("log_blob");
  EXPECT_EQ(batch.Count(), 1);  // LogData doesn't count

  WriteBatchTestHandler handler;
  ASSERT_TRUE(batch.Iterate(&handler).ok());
  ASSERT_EQ(handler.entries.size(), 1u);
  ASSERT_EQ(handler.log_data.size(), 1u);
  EXPECT_EQ(handler.log_data[0], "log_blob");
}

TEST(WriteBatchTest, Clear) {
  WriteBatch batch;
  batch.Put("k", "v");
  EXPECT_EQ(batch.Count(), 1);
  batch.Clear();
  EXPECT_EQ(batch.Count(), 0);
  EXPECT_FALSE(batch.HasPut());
}

TEST(WriteBatchTest, LargeBatch) {
  WriteBatch batch;
  for (int i = 0; i < 10000; i++) {
    std::string k = "key_" + std::to_string(i);
    std::string v = "val_" + std::to_string(i);
    batch.Put(k, v);
  }
  EXPECT_EQ(batch.Count(), 10000);

  WriteBatchTestHandler handler;
  ASSERT_TRUE(batch.Iterate(&handler).ok());
  ASSERT_EQ(handler.entries.size(), 10000u);
}

TEST(WriteBatchTest, Append) {
  WriteBatch a, b;
  a.Put("a1", "v1");
  b.Put("b1", "v2");
  b.Delete("b2");

  WriteBatchInternal::Append(&a, &b);
  EXPECT_EQ(a.Count(), 3);

  WriteBatchTestHandler handler;
  ASSERT_TRUE(a.Iterate(&handler).ok());
  ASSERT_EQ(handler.entries.size(), 3u);
  EXPECT_EQ(handler.entries[0].key, "a1");
  EXPECT_EQ(handler.entries[1].key, "b1");
  EXPECT_EQ(handler.entries[2].key, "b2");
}

TEST(WriteBatchTest, SetContents) {
  WriteBatch orig;
  orig.Put("k1", "v1");
  orig.SetSequence(999);

  WriteBatch copy;
  WriteBatchInternal::SetContents(&copy,
                                  WriteBatchInternal::Contents(&orig));
  EXPECT_EQ(copy.Count(), 1);
  EXPECT_EQ(copy.Sequence(), 999u);

  WriteBatchTestHandler handler;
  ASSERT_TRUE(copy.Iterate(&handler).ok());
  EXPECT_EQ(handler.entries[0].key, "k1");
}

}  // namespace mwal
