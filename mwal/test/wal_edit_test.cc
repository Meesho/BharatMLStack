#include <gtest/gtest.h>

#include "mwal/slice.h"
#include "mwal/status.h"
#include "wal/wal_edit.h"

namespace mwal {

TEST(WalEditTest, AdditionRoundtrip) {
  WalAddition orig(42, 12345, true);
  std::string encoded;
  orig.EncodeTo(&encoded);

  WalAddition decoded;
  Slice input(encoded);
  ASSERT_TRUE(decoded.DecodeFrom(&input).ok());
  EXPECT_EQ(decoded.log_number, 42u);
  EXPECT_EQ(decoded.size_bytes, 12345u);
  EXPECT_TRUE(decoded.synced);
}

TEST(WalEditTest, DeletionRoundtrip) {
  WalDeletion orig(99);
  std::string encoded;
  orig.EncodeTo(&encoded);

  WalDeletion decoded;
  Slice input(encoded);
  ASSERT_TRUE(decoded.DecodeFrom(&input).ok());
  EXPECT_EQ(decoded.log_number, 99u);
}

TEST(WalEditTest, WalSetAddAndDelete) {
  WalSet ws;
  ws.AddWal(WalAddition(1, 100, true));
  ws.AddWal(WalAddition(2, 200, false));
  ws.AddWal(WalAddition(5, 500, true));

  EXPECT_TRUE(ws.HasWal(1));
  EXPECT_TRUE(ws.HasWal(2));
  EXPECT_TRUE(ws.HasWal(5));
  EXPECT_FALSE(ws.HasWal(3));

  EXPECT_EQ(ws.GetMinLogNumber(), 1u);

  ws.DeleteWalsBefore(3);
  EXPECT_FALSE(ws.HasWal(1));
  EXPECT_FALSE(ws.HasWal(2));
  EXPECT_TRUE(ws.HasWal(5));
  EXPECT_EQ(ws.GetMinLogNumber(), 5u);
}

TEST(WalEditTest, WalSetEmpty) {
  WalSet ws;
  EXPECT_EQ(ws.GetMinLogNumber(), 0u);
  EXPECT_TRUE(ws.CheckWals().ok());
}

TEST(WalEditTest, DebugString) {
  WalAddition add(10, 5000, false);
  std::string s = add.DebugString();
  EXPECT_NE(s.find("10"), std::string::npos);
  EXPECT_NE(s.find("5000"), std::string::npos);
}

}  // namespace mwal
