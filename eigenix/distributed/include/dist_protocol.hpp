#ifndef EIGENIX_DIST_PROTOCOL_HPP
#define EIGENIX_DIST_PROTOCOL_HPP

#include <cstdint>

namespace eigenix::dist {

enum class MsgType : uint32_t {
    // Coordinator → Worker
    SHARD_CONFIG   = 0x01,
    SHARD_DATA     = 0x02,
    CENTROIDS      = 0x03,
    DONE           = 0x04,

    // Worker → Coordinator
    READY          = 0x10,
    LOCAL_STATS    = 0x11,

    ERROR          = 0xFF,
};

static constexpr uint32_t PROTOCOL_MAGIC = 0xE16E1E16;

struct alignas(4) MsgHeader {
    uint32_t magic;
    uint32_t msg_type;
    uint32_t payload_len;
};
static_assert(sizeof(MsgHeader) == 12, "MsgHeader must be 12 bytes");

struct ShardConfig {
    uint32_t n_local;
    uint32_t dim;
    uint32_t k;
    uint32_t max_iter;
    float    tol;
    uint32_t seed;
    uint32_t worker_id;
    uint32_t n_workers;
};
static_assert(sizeof(ShardConfig) == 32, "ShardConfig must be 32 bytes");

}  // namespace eigenix::dist

#endif  // EIGENIX_DIST_PROTOCOL_HPP
