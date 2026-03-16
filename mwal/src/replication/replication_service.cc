#include "replication/replication_service.h"

#include "mwal/wal_iterator.h"
#include "mwal/write_batch.h"
#include "replication/replication_manager.h"

namespace mwal {
namespace replication {

ReplicationServiceImpl::ReplicationServiceImpl(DBWal* wal,
                                               ReplicationManager* mgr)
    : wal_(wal), mgr_(mgr) {}

grpc::Status ReplicationServiceImpl::Replicate(
    grpc::ServerContext* /*ctx*/, const ReplicateRequest* req,
    ReplicateResponse* resp) {
  uint64_t my_term = mgr_->GetCurrentTerm();
  resp->set_term_seen(my_term);

  // Reject stale leaders.
  if (req->term() < my_term) {
    resp->set_success(false);
    resp->set_message("stale term");
    resp->set_last_persisted_lsn(wal_->GetLatestSequenceNumber());
    return grpc::Status::OK;
  }

  uint64_t my_lsn = wal_->GetLatestSequenceNumber();

  // Gap / divergence detection via prev_lsn.
  if (req->prev_lsn() > my_lsn) {
    // Gap: replica is behind; tell leader so it can use StreamWAL.
    resp->set_success(false);
    resp->set_message("gap detected");
    resp->set_last_persisted_lsn(my_lsn);
    return grpc::Status::OK;
  }
  if (req->prev_lsn() < my_lsn) {
    // Divergent tail: truncate to prev_lsn.
    auto s = wal_->TruncateAfter(req->prev_lsn());
    if (!s.ok()) {
      resp->set_success(false);
      resp->set_message(std::string("truncate failed: ") +
                        s.ToString());
      resp->set_last_persisted_lsn(wal_->GetLatestSequenceNumber());
      return grpc::Status::OK;
    }
  }

  // Append all entries.
  for (const auto& entry : req->entries()) {
    Slice payload(entry.payload().data(), entry.payload().size());
    auto s = wal_->AppendReplicated(
        entry.first_seq(), entry.count(), payload, req->term());
    if (!s.ok()) {
      resp->set_success(false);
      resp->set_message(std::string("append failed: ") +
                        s.ToString());
      resp->set_last_persisted_lsn(wal_->GetLatestSequenceNumber());
      return grpc::Status::OK;
    }
  }

  resp->set_success(true);
  resp->set_last_persisted_lsn(wal_->GetLatestSequenceNumber());
  return grpc::Status::OK;
}

grpc::Status ReplicationServiceImpl::StreamWAL(
    grpc::ServerContext* /*ctx*/, const StreamWALRequest* req,
    grpc::ServerWriter<WALChunk>* writer) {
  std::unique_ptr<WalIterator> iter;
  auto s = wal_->NewWalIterator(req->start_lsn(), &iter);
  if (!s.ok()) {
    return grpc::Status(grpc::INTERNAL,
                        std::string("iterator failed: ") + s.ToString());
  }

  WALChunk chunk;
  uint32_t entries_in_chunk = 0;
  const uint32_t kMaxChunkEntries = 100;

  while (iter->Valid()) {
    const WriteBatch& batch = iter->GetBatch();
    auto* entry = chunk.add_entries();
    entry->set_first_seq(iter->GetSequenceNumber());
    entry->set_count(static_cast<uint32_t>(batch.Count()));
    entry->set_payload(batch.Data());

    entries_in_chunk++;

    if (req->end_lsn() > 0 &&
        iter->GetSequenceNumber() >= req->end_lsn()) {
      break;
    }

    if (entries_in_chunk >= kMaxChunkEntries) {
      writer->Write(chunk);
      chunk.Clear();
      entries_in_chunk = 0;
    }

    iter->Next();
  }

  // Flush remaining.
  if (entries_in_chunk > 0) {
    writer->Write(chunk);
  }

  return grpc::Status::OK;
}

grpc::Status ReplicationServiceImpl::ReportProgress(
    grpc::ServerContext* /*ctx*/, const ProgressReport* req,
    ProgressAck* resp) {
  mgr_->HandleProgressReport(req->node_id(), req->persisted_lsn(),
                              req->applied_lsn());
  resp->set_committed_lsn(mgr_->GetCommittedLSN());
  return grpc::Status::OK;
}

}  // namespace replication
}  // namespace mwal
