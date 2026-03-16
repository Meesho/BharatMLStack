#pragma once

#include <grpcpp/grpcpp.h>

#include "mwal/db_wal.h"
#include "replication.grpc.pb.h"
#include "replication/isr_tracker.h"

namespace mwal {
namespace replication {

class ReplicationManager;  // forward decl to avoid circular include

// gRPC server-side implementation running on every node.
// On replicas: handles Replicate and StreamWAL from the leader.
// On leader: handles ReportProgress from followers.
class ReplicationServiceImpl final
    : public mwal::replication::ReplicationService::Service {
 public:
  ReplicationServiceImpl(DBWal* wal, ReplicationManager* mgr);

  grpc::Status Replicate(grpc::ServerContext* ctx,
                         const ReplicateRequest* req,
                         ReplicateResponse* resp) override;

  grpc::Status StreamWAL(grpc::ServerContext* ctx,
                         const StreamWALRequest* req,
                         grpc::ServerWriter<WALChunk>* writer) override;

  grpc::Status ReportProgress(grpc::ServerContext* ctx,
                               const ProgressReport* req,
                               ProgressAck* resp) override;

 private:
  DBWal* wal_;
  ReplicationManager* mgr_;
};

}  // namespace replication
}  // namespace mwal
