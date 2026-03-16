#pragma once

#include <memory>
#include <string>

#include <grpcpp/grpcpp.h>

#include "replication.grpc.pb.h"

namespace mwal {
namespace replication {

// gRPC client stub for communicating with a single remote replica.
class ReplicationClient {
 public:
  explicit ReplicationClient(const std::string& target_endpoint);

  // Send a Replicate RPC (synchronous).
  ReplicateResponse SendReplicate(const ReplicateRequest& req);

  // Initiate catch-up: stream WAL entries from the remote node.
  // Calls |on_chunk| for each chunk received.  Returns true on success.
  bool RequestStreamWAL(
      const StreamWALRequest& req,
      std::function<bool(const WALChunk& chunk)> on_chunk);

  // Send a progress report (follower → leader).
  ProgressAck SendProgressReport(const ProgressReport& req);

  bool IsConnected() const;

 private:
  std::shared_ptr<grpc::Channel> channel_;
  std::unique_ptr<mwal::replication::ReplicationService::Stub> stub_;
};

}  // namespace replication
}  // namespace mwal
