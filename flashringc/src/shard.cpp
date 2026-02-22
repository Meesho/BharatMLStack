#include "shard.h"
#include "aligned_buffer.h"

#include <algorithm>
#include <chrono>
#include <cstring>

Shard::Shard(RingDevice ring, size_t mt_size, uint32_t index_cap,
             bool use_io_uring)
    : ring_(std::move(ring)),
      memtables_(ring_, mt_size),
      index_(index_cap),
      delete_mgr_(ring_, index_),
      io_engine_(use_io_uring) {}

bool Shard::put(Hash128 h, const void* rec, size_t rec_len) {
    WriteResult wr = memtables_.put(rec, rec_len);
    {
        std::unique_lock lk(mu_);
        index_.put(h, wr.mem_id, wr.offset, wr.length);
        delete_mgr_.maybe_evict();
    }
    delete_mgr_.flush_discards();
    return true;
}

bool Shard::get(Hash128 h, const void* key, size_t key_len,
                void* val_buf, size_t val_buf_len, size_t* actual_len) {
    LookupResult lr{};
    {
        auto now   = std::chrono::steady_clock::now().time_since_epoch();
        auto delta = static_cast<uint16_t>(
            std::chrono::duration_cast<std::chrono::seconds>(now).count() & 0xFFFF);
        std::shared_lock lk(mu_);
        if (!index_.get(h, delta, lr))
            return false;
    }

    std::vector<uint8_t> buf(lr.length);
    if (!read_record(lr, buf))
        return false;

    DecodedRecord dr = decode_record(buf.data());

    if (dr.key_len != key_len ||
        std::memcmp(dr.key, key, key_len) != 0)
        return false;

    if (actual_len)
        *actual_len = dr.val_len;

    size_t copy_len = std::min(static_cast<size_t>(dr.val_len), val_buf_len);
    if (copy_len > 0)
        std::memcpy(val_buf, dr.val, copy_len);

    return true;
}

void Shard::get_batch(const BatchGetRequest* reqs, BatchGetResult* results,
                      size_t count, const Hash128* hashes) {
    if (count == 0) return;

    auto now   = std::chrono::steady_clock::now().time_since_epoch();
    auto delta = static_cast<uint16_t>(
        std::chrono::duration_cast<std::chrono::seconds>(now).count() & 0xFFFF);

    struct PendingRead {
        size_t        req_idx;
        LookupResult  lr;
        int64_t       file_off;
        size_t        aligned_len;
        AlignedBuffer abuf;
    };

    std::vector<LookupResult> lrs(count);
    std::vector<bool>         found_in_index(count, false);
    std::vector<std::vector<uint8_t>> mem_bufs(count);

    // Phase 1: index lookup + memtable reads under shared lock
    {
        std::shared_lock lk(mu_);
        for (size_t i = 0; i < count; ++i) {
            if (index_.get(hashes[i], delta, lrs[i])) {
                found_in_index[i] = true;
                mem_bufs[i].resize(lrs[i].length);
                if (memtables_.try_read_from_memory(
                        mem_bufs[i].data(), lrs[i].length,
                        lrs[i].mem_id, lrs[i].offset)) {
                    // Served from memtable — decode and verify
                    DecodedRecord dr = decode_record(mem_bufs[i].data());
                    if (dr.key_len == reqs[i].key_len &&
                        std::memcmp(dr.key, reqs[i].key, dr.key_len) == 0) {
                        results[i].found = true;
                        results[i].actual_len = dr.val_len;
                        size_t cpy = std::min(static_cast<size_t>(dr.val_len),
                                              reqs[i].val_buf_len);
                        if (cpy > 0)
                            std::memcpy(reqs[i].val_buf, dr.val, cpy);
                    }
                    found_in_index[i] = false;  // already handled
                }
            }
        }
    }

    // Phase 2: collect disk reads
    std::vector<PendingRead> pending;
    for (size_t i = 0; i < count; ++i) {
        if (!found_in_index[i]) continue;

        int64_t file_off = memtables_.file_offset_for(lrs[i].mem_id);
        if (file_off < 0) continue;

        size_t aligned_len = AlignedBuffer::align_up(lrs[i].length);
        auto abuf = AlignedBuffer::allocate(aligned_len);
        if (!abuf.valid()) continue;

        pending.push_back({i, lrs[i], file_off, aligned_len, std::move(abuf)});
    }

    if (pending.empty()) return;

    // Phase 3: submit all disk reads via IoEngine
    std::vector<ReadOp> ops(pending.size());
    for (size_t j = 0; j < pending.size(); ++j) {
        auto& p = pending[j];
        uint64_t abs_offset = static_cast<uint64_t>(p.file_off) +
                              p.lr.offset + ring_.base_offset();
        ops[j] = {
            .buf    = p.abuf.data(),
            .len    = p.aligned_len,
            .offset = abs_offset,
            .fd     = ring_.read_fd(),
        };
    }

    io_engine_.read_batch(ops.data(), static_cast<uint32_t>(ops.size()));

    // Phase 4: decode results
    for (size_t j = 0; j < pending.size(); ++j) {
        if (!ops[j].ok) continue;

        auto& p = pending[j];
        DecodedRecord dr = decode_record(p.abuf.bytes());

        if (dr.key_len != reqs[p.req_idx].key_len ||
            std::memcmp(dr.key, reqs[p.req_idx].key, dr.key_len) != 0)
            continue;

        results[p.req_idx].found = true;
        results[p.req_idx].actual_len = dr.val_len;
        size_t cpy = std::min(static_cast<size_t>(dr.val_len),
                              reqs[p.req_idx].val_buf_len);
        if (cpy > 0)
            std::memcpy(reqs[p.req_idx].val_buf, dr.val, cpy);
    }
}

bool Shard::remove(Hash128 h) {
    std::unique_lock lk(mu_);
    return index_.remove(h);
}

void Shard::flush() {
    memtables_.flush();
}

uint32_t Shard::key_count() const {
    std::shared_lock lk(mu_);
    return index_.size();
}

uint64_t Shard::ring_usage() const {
    return ring_.write_offset();
}

bool Shard::read_record(const LookupResult& lr, std::vector<uint8_t>& buf) {
    buf.resize(lr.length);

    if (memtables_.try_read_from_memory(buf.data(), lr.length,
                                         lr.mem_id, lr.offset))
        return true;

    int64_t file_off = memtables_.file_offset_for(lr.mem_id);
    if (file_off < 0)
        return false;

    uint64_t abs_offset = static_cast<uint64_t>(file_off) + lr.offset +
                          ring_.base_offset();
    size_t aligned_len = AlignedBuffer::align_up(lr.length);
    auto tmp = AlignedBuffer::allocate(aligned_len);

    ReadOp op = {
        .buf    = tmp.data(),
        .len    = aligned_len,
        .offset = abs_offset,
        .fd     = ring_.read_fd(),
    };
    io_engine_.read_batch(&op, 1);

    if (!op.ok)
        return false;
    std::memcpy(buf.data(), tmp.bytes(), lr.length);
    return true;
}
