#pragma once

#include "flashringc/common.h"

#include <atomic>
#include <cstdint>
#include <string_view>
#include <vector>

// Forward declarations to avoid circular includes.
struct LookupResult;

// ---------------------------------------------------------------------------
// Result — written to caller-owned pointer on completion
// ---------------------------------------------------------------------------

struct Result {
    Status                 status = Status::Error;
    std::vector<uint8_t>   value;
};

// ---------------------------------------------------------------------------
// KVPair — used by batch_put
// ---------------------------------------------------------------------------

struct KVPair {
    std::string_view key;
    std::string_view value;
};

// ---------------------------------------------------------------------------
// Request — submitted to a ShardReactor via MPSC queue
// Caller keeps key/value and result memory alive until sem_wait() returns.
// ---------------------------------------------------------------------------

enum class OpType : uint8_t { Get, Put, Delete, Shutdown };

struct Request {
    OpType                  type;
    Hash128                 hash;    // pre-computed by Cache layer
    std::string_view        key;
    std::string_view        value;   // populated for Put
    Result*                 result          = nullptr;
    uint32_t                sem_slot        = 0;
    std::atomic<int>*       batch_remaining = nullptr;

    Request() = default;
    Request(Request&&) = default;
    Request& operator=(Request&&) = default;
    Request(const Request&) = delete;
    Request& operator=(const Request&) = delete;
};

// ---------------------------------------------------------------------------
// PendingOp — tracks in-flight io_uring operations inside ShardReactor
// ---------------------------------------------------------------------------

enum class PendingOpType : uint8_t { GetRead, Flush, Trim };

struct PendingOp {
    PendingOpType   type;
    AlignedBuffer   buf;         // IO buffer (owned)
    Request*        request;     // non-owning; for GetRead → resolve promise
    uint32_t        mem_id;      // memtable generation (GetRead: source; Flush: flushed)
    uint32_t        offset;      // record offset within memtable (GetRead)
    uint32_t        length;      // record length (GetRead)
    int             flushed_idx; // which memtable slot was flushed (Flush)
    int64_t         write_off;   // ring offset where flush was written (Flush)

    PendingOp() = default;
    PendingOp(PendingOp&&) = default;
    PendingOp& operator=(PendingOp&&) = default;
    PendingOp(const PendingOp&) = delete;
    PendingOp& operator=(const PendingOp&) = delete;
};
