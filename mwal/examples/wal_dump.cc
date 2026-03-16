// wal_dump: Read a WAL directory and output all records as deterministic text.
//
// Usage:
//   wal_dump <wal_dir> [start_seq]
//
// Output format (one line per record in the batch):
//   SEQ=<seq> COUNT=<count> PUT <key> <value>
//   SEQ=<seq> COUNT=<count> DELETE <key>
//
// Used by integration test scripts to diff WAL contents across nodes.

#include <cstdlib>
#include <iostream>
#include <memory>
#include <string>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/slice.h"
#include "mwal/status.h"
#include "mwal/wal_iterator.h"
#include "mwal/write_batch.h"

namespace {

class DumpHandler : public mwal::WriteBatch::Handler {
 public:
  DumpHandler(mwal::SequenceNumber seq, int count)
      : seq_(seq), count_(count) {}

  mwal::Status Put(const mwal::Slice& key, const mwal::Slice& value) override {
    std::cout << "SEQ=" << seq_ << " COUNT=" << count_
              << " PUT " << key.ToString() << " " << value.ToString() << "\n";
    return mwal::Status::OK();
  }

  mwal::Status Delete(const mwal::Slice& key) override {
    std::cout << "SEQ=" << seq_ << " COUNT=" << count_
              << " DELETE " << key.ToString() << "\n";
    return mwal::Status::OK();
  }

 private:
  mwal::SequenceNumber seq_;
  int count_;
};

}  // namespace

int main(int argc, char** argv) {
  if (argc < 2) {
    std::cerr << "Usage: " << argv[0] << " <wal_dir> [start_seq]\n";
    return 1;
  }

  std::string wal_dir = argv[1];
  mwal::SequenceNumber start_seq = 0;
  if (argc >= 3) {
    start_seq = std::stoull(argv[2]);
  }

  mwal::WALOptions opts;
  opts.wal_dir = wal_dir;

  std::unique_ptr<mwal::DBWal> wal;
  auto s = mwal::DBWal::Open(opts, mwal::Env::Default(), &wal);
  if (!s.ok()) {
    std::cerr << "Failed to open WAL: " << s.ToString() << "\n";
    return 1;
  }

  std::unique_ptr<mwal::WalIterator> it;
  s = wal->NewWalIterator(start_seq, &it);
  if (!s.ok()) {
    std::cerr << "Failed to create iterator: " << s.ToString() << "\n";
    wal->Close();
    return 1;
  }

  uint64_t batch_count = 0;
  for (; it->Valid(); it->Next()) {
    const auto& batch = it->GetBatch();
    auto seq = it->GetSequenceNumber();
    DumpHandler handler(seq, batch.Count());
    s = batch.Iterate(&handler);
    if (!s.ok()) {
      std::cerr << "Iterate error at seq=" << seq << ": " << s.ToString()
                << "\n";
      break;
    }
    batch_count++;
  }

  if (!it->status().ok()) {
    std::cerr << "Iterator error: " << it->status().ToString() << "\n";
  }

  std::cerr << "Dumped " << batch_count << " batches from " << wal_dir << "\n";

  wal->Close();
  return 0;
}
