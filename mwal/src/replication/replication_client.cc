#include "replication/replication_client.h"

namespace mwal {
namespace replication {

ReplicationClient::ReplicationClient(const std::string& target_endpoint)
    : channel_(grpc::CreateChannel(target_endpoint,
                                   grpc::InsecureChannelCredentials())),
      stub_(mwal::replication::ReplicationService::NewStub(channel_)) {}

ReplicateResponse ReplicationClient::SendReplicate(
    const ReplicateRequest& req) {
  grpc::ClientContext ctx;
  ctx.set_deadline(std::chrono::system_clock::now() +
                   std::chrono::milliseconds(5000));
  ReplicateResponse resp;
  grpc::Status st = stub_->Replicate(&ctx, req, &resp);
  if (!st.ok()) {
    resp.set_success(false);
    resp.set_message(st.error_message());
  }
  return resp;
}

bool ReplicationClient::RequestStreamWAL(
    const StreamWALRequest& req,
    std::function<bool(const WALChunk& chunk)> on_chunk) {
  grpc::ClientContext ctx;
  auto reader = stub_->StreamWAL(&ctx, req);
  WALChunk chunk;
  while (reader->Read(&chunk)) {
    if (!on_chunk(chunk)) break;
  }
  grpc::Status st = reader->Finish();
  return st.ok();
}

ProgressAck ReplicationClient::SendProgressReport(
    const ProgressReport& req) {
  grpc::ClientContext ctx;
  ctx.set_deadline(std::chrono::system_clock::now() +
                   std::chrono::milliseconds(2000));
  ProgressAck resp;
  stub_->ReportProgress(&ctx, req, &resp);
  return resp;
}

bool ReplicationClient::IsConnected() const {
  return channel_ &&
         channel_->GetState(false) != GRPC_CHANNEL_SHUTDOWN;
}

}  // namespace replication
}  // namespace mwal
