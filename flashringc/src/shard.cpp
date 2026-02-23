#include "shard.h"
#include "aligned_buffer.h"

#include <algorithm>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <thread>
#include <unordered_map>

// ---------------------------------------------------------------------------
// BatchCompletion
// ---------------------------------------------------------------------------

void BatchCompletion::mark_one_done() {
    std::lock_guard<std::mutex> lock(mu_);
    if (remaining_ > 0) --remaining_;
    if (remaining_ == 0) cv_.notify_one();
}

void BatchCompletion::wait() {
    std::unique_lock<std::mutex> lock(mu_);
    cv_.wait(lock, [this] { return remaining_ == 0; });
}

// ---------------------------------------------------------------------------
// Wide-read constants / debug
// ---------------------------------------------------------------------------

static constexpr size_t kWideReadBlockSize = 128 * 1024;

// Set to true (or use getenv) to log reactor/put activity to stderr
static bool reactor_log_enabled() {
    static bool once = []() {
        const char* e = std::getenv("FLASHRING_REACTOR_LOG");
        return e && (e[0] == '1' || e[0] == 'y' || e[0] == 'Y');
    }();
    return once;
}
#define REACT_LOG(fmt, ...) \
    do { if (reactor_log_enabled()) std::fprintf(stderr, "[reactor] " fmt "\n", ##__VA_ARGS__); } while (0)
#define PUT_LOG(fmt, ...) \
    do { if (reactor_log_enabled()) std::fprintf(stderr, "[put] " fmt "\n", ##__VA_ARGS__); } while (0)

// ---------------------------------------------------------------------------
// Shard ctor/dtor
// ---------------------------------------------------------------------------

Shard::Shard(RingDevice ring, size_t mt_size, uint32_t index_cap,
             bool use_io_uring)
    : ring_(std::move(ring)),
      memtables_(ring_, mt_size),
      index_(index_cap),
      delete_mgr_(ring_, index_),
      io_engine_(use_io_uring) {
    reactor_thread_ = std::thread(&Shard::reactor_loop, this);
}

Shard::~Shard() {
    {
        std::lock_guard<std::mutex> lock(queue_mu_);
        shutdown_ = true;
    }
    queue_cv_.notify_one();
    if (reactor_thread_.joinable())
        reactor_thread_.join();
}

bool Shard::put(Hash128 h, const void* rec, size_t rec_len) {
    PUT_LOG("enter rec_len=%zu", rec_len);
    WriteResult wr = memtables_.put(rec, rec_len);
    {
        std::unique_lock lk(mu_);
        index_.put(h, wr.mem_id, wr.offset, wr.length);
        delete_mgr_.maybe_evict();
    }
    delete_mgr_.flush_discards();
    PUT_LOG("done");
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

    if (kRecordHeaderSize + dr.key_len + dr.val_len > lr.length)
        return false;
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
                    DecodedRecord dr = decode_record(mem_bufs[i].data());
                    if (kRecordHeaderSize + dr.key_len + dr.val_len > lrs[i].length)
                        continue;
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

    // Phase 2: collect disk reads (with ring-region bounds check)
    std::vector<PendingRead> pending;
    for (size_t i = 0; i < count; ++i) {
        if (!found_in_index[i]) continue;

        int64_t file_off = memtables_.file_offset_for(lrs[i].mem_id);
        if (file_off < 0) continue;

        uint64_t ring_offset = static_cast<uint64_t>(file_off) + lrs[i].offset;
        size_t aligned_len = AlignedBuffer::align_up(lrs[i].length);
        if (ring_offset + aligned_len > ring_.capacity()) continue;

        auto abuf = AlignedBuffer::allocate(aligned_len);
        if (!abuf.valid()) continue;

        pending.push_back({i, lrs[i], file_off, aligned_len, std::move(abuf)});
    }

    if (pending.empty()) return;

    // Phase 3: submit all disk reads via IoEngine
    std::vector<ReadOp> ops(pending.size());
    for (size_t j = 0; j < pending.size(); ++j) {
        auto& p = pending[j];
        uint64_t ring_off = static_cast<uint64_t>(p.file_off) + p.lr.offset;
        ops[j] = {
            .buf    = p.abuf.data(),
            .len    = p.aligned_len,
            .offset = ring_off + ring_.base_offset(),
            .fd     = ring_.read_fd(),
        };
    }

    io_engine_.read_batch(ops.data(), static_cast<uint32_t>(ops.size()));

    // Phase 4: decode results
    for (size_t j = 0; j < pending.size(); ++j) {
        if (!ops[j].ok) continue;

        auto& p = pending[j];
        DecodedRecord dr = decode_record(p.abuf.bytes());

        if (kRecordHeaderSize + dr.key_len + dr.val_len > p.lr.length)
            continue;
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

void Shard::submit_batch(ShardBatchRequest req) {
    if (req.count == 0) {
        if (req.completion) req.completion->mark_one_done();
        return;
    }
    std::lock_guard<std::mutex> lock(queue_mu_);
    request_queue_.push(std::move(req));
    queue_cv_.notify_one();
}

// ---------------------------------------------------------------------------
// Reactor loop: drain queue, index+memtable, coalesce into wide reads, submit/reap
// ---------------------------------------------------------------------------

namespace {

struct PendingDiskEntry {
    uint64_t           ring_offset;
    uint32_t           record_len;
    BatchGetResult*    result;
    const void*        key;
    size_t             key_len;
    void*              val_buf;
    size_t             val_buf_len;
};

struct DecodeEntry {
    size_t             offset_in_buf;
    uint32_t           record_len;
    BatchGetResult*    result;
    const void*        key;
    size_t             key_len;
    void*              val_buf;
    size_t             val_buf_len;
};

struct InFlightRead {
    ReadOp                  op;
    AlignedBuffer           buf;
    std::vector<DecodeEntry> entries;
    BatchCompletion*        completion = nullptr;  // mark_one_done when this read is reaped (and no others for same batch)
};

}  // namespace

void Shard::reactor_loop() {
    std::vector<PendingDiskEntry> pending_disk;
    pending_disk.reserve(256);
    std::vector<InFlightRead> in_flight;
    in_flight.reserve(32);

    while (true) {
        // Reap any ready completions (non-blocking); sets op->ok for completed ops
        if (io_engine_.uring_enabled()) {
            uint32_t reaped = io_engine_.reap_ready(0);
            if (reaped > 0) REACT_LOG("reap reaped=%u in_flight=%zu", reaped, in_flight.size());
            // Scan in_flight for completed ops (op.ok set by reap)
            for (size_t i = 0; i < in_flight.size(); ) {
                if (!in_flight[i].op.ok) { ++i; continue; }
                InFlightRead& r = in_flight[i];
                for (const auto& e : r.entries) {
                    const uint8_t* ptr = r.buf.bytes() + e.offset_in_buf;
                    if (e.offset_in_buf + e.record_len > r.buf.size()) continue;
                    DecodedRecord dr = decode_record(ptr);
                    if (kRecordHeaderSize + dr.key_len + dr.val_len > e.record_len) continue;
                    if (dr.key_len != e.key_len || std::memcmp(dr.key, e.key, dr.key_len) != 0) continue;
                    e.result->found = true;
                    e.result->actual_len = dr.val_len;
                    size_t cpy = std::min(static_cast<size_t>(dr.val_len), e.val_buf_len);
                    if (cpy > 0) std::memcpy(e.val_buf, dr.val, cpy);
                }
                BatchCompletion* comp = in_flight[i].completion;
                in_flight[i] = std::move(in_flight.back());
                in_flight.pop_back();
                if (comp) {
                    bool any_left = false;
                    for (const auto& r : in_flight)
                        if (r.completion == comp) { any_left = true; break; }
                    if (!any_left) {
                        REACT_LOG("mark_one_done (batch slice complete)");
                        comp->mark_one_done();
                    }
                }
            }
        }

        // Drain one request from queue; don't block when in_flight (need to reap first)
        ShardBatchRequest batch;
        {
            std::unique_lock<std::mutex> lock(queue_mu_);
            for (;;) {
                if (!request_queue_.empty()) {
            batch = std::move(request_queue_.front());
            request_queue_.pop();
            REACT_LOG("drain count=%zu queue_sz=%zu", batch.count, request_queue_.size());
            break;
        }
        if (shutdown_) return;
                if (!in_flight.empty()) {
                    REACT_LOG("queue empty in_flight=%zu yield", in_flight.size());
                    lock.unlock();
                    std::this_thread::sleep_for(std::chrono::microseconds(50));
                    lock.lock();
                    continue;  // re-check queue; top of loop will reap
                }
                queue_cv_.wait(lock, [this] { return shutdown_ || !request_queue_.empty(); });
            }
        }

        const BatchGetRequest* reqs = batch.reqs;
        const Hash128* hashes = batch.hashes;
        BatchGetResult* results_base = batch.results;
        const size_t* result_indices = batch.result_indices;
        const size_t count = batch.count;
        BatchCompletion* completion = batch.completion;
        auto result_at = [&](size_t i) -> BatchGetResult* {
            return result_indices ? &results_base[result_indices[i]] : &results_base[i];
        };

        auto now = std::chrono::steady_clock::now().time_since_epoch();
        auto delta = static_cast<uint16_t>(
            std::chrono::duration_cast<std::chrono::seconds>(now).count() & 0xFFFF);

        std::vector<LookupResult> lrs(count);
        std::vector<bool> found_in_index(count, false);

        // Index + memtable under shared lock
        {
            std::shared_lock lk(mu_);
            for (size_t i = 0; i < count; ++i) {
                if (!index_.get(hashes[i], delta, lrs[i])) continue;
                found_in_index[i] = true;
                std::vector<uint8_t> mem_buf(lrs[i].length);
                if (memtables_.try_read_from_memory(
                        mem_buf.data(), lrs[i].length, lrs[i].mem_id, lrs[i].offset)) {
                    DecodedRecord dr = decode_record(mem_buf.data());
                    if (kRecordHeaderSize + dr.key_len + dr.val_len <= lrs[i].length &&
                        dr.key_len == reqs[i].key_len &&
                        std::memcmp(dr.key, reqs[i].key, dr.key_len) == 0) {
                        BatchGetResult* r = result_at(i);
                        r->found = true;
                        r->actual_len = dr.val_len;
                        size_t cpy = std::min(static_cast<size_t>(dr.val_len), reqs[i].val_buf_len);
                        if (cpy > 0) std::memcpy(reqs[i].val_buf, dr.val, cpy);
                        found_in_index[i] = false;
                    }
                }
            }
        }

        // Build pending_disk for misses
        for (size_t i = 0; i < count; ++i) {
            if (!found_in_index[i]) continue;
            int64_t file_off = memtables_.file_offset_for(lrs[i].mem_id);
            if (file_off < 0) continue;
            uint64_t ring_offset = static_cast<uint64_t>(file_off) + lrs[i].offset;
            size_t aligned_len = AlignedBuffer::align_up(lrs[i].length);
            if (ring_offset + aligned_len > ring_.capacity()) continue;
            pending_disk.push_back({
                ring_offset,
                lrs[i].length,
                result_at(i),
                reqs[i].key,
                reqs[i].key_len,
                reqs[i].val_buf,
                reqs[i].val_buf_len,
            });
        }

        bool should_flush = !pending_disk.empty();

        if (should_flush && io_engine_.uring_enabled()) {
            // Merge into aligned blocks
            std::unordered_map<uint64_t, std::vector<size_t>> block_to_entries;
            for (size_t j = 0; j < pending_disk.size(); ++j) {
                uint64_t block_start = (pending_disk[j].ring_offset / kWideReadBlockSize) * kWideReadBlockSize;
                block_to_entries[block_start].push_back(j);
            }
            uint64_t base = ring_.base_offset();
            int fd = ring_.read_fd();
            uint64_t cap = ring_.capacity();

            for (const auto& [block_start, indices] : block_to_entries) {
                size_t read_len = std::min(kWideReadBlockSize, static_cast<size_t>(cap - block_start));
                if (read_len == 0) continue;
                read_len = AlignedBuffer::align_up(read_len);
                auto buf = AlignedBuffer::allocate(read_len);
                if (!buf.valid()) continue;

                InFlightRead in;
                in.op.buf = buf.data();
                in.op.len = read_len;
                in.op.offset = base + block_start;
                in.op.fd = fd;
                in.op.ok = false;
                in.buf = std::move(buf);

                for (size_t j : indices) {
                    const auto& e = pending_disk[j];
                    size_t offset_in_buf = e.ring_offset - block_start;
                in.entries.push_back({
                    offset_in_buf,
                    e.record_len,
                    e.result,
                    e.key,
                    e.key_len,
                    e.val_buf,
                    e.val_buf_len,
                });
            }
            in.completion = completion;
            in_flight.push_back(std::move(in));
        }
            // Submit each new in_flight op (kernel needs real op* for completion)
            size_t submit_start = in_flight.size() - block_to_entries.size();
            for (size_t i = submit_start; i < in_flight.size(); ++i)
                io_engine_.submit_only(&in_flight[i].op, 1);
            REACT_LOG("submit_uring blocks=%zu in_flight=%zu", block_to_entries.size(), in_flight.size());
            pending_disk.clear();
        } else if (should_flush && !io_engine_.uring_enabled()) {
            // Fallback: small reads with blocking read_batch
            std::vector<ReadOp> ops(pending_disk.size());
            std::vector<AlignedBuffer> bufs(pending_disk.size());
            for (size_t j = 0; j < pending_disk.size(); ++j) {
                size_t aligned_len = AlignedBuffer::align_up(pending_disk[j].record_len);
                bufs[j] = AlignedBuffer::allocate(aligned_len);
                if (!bufs[j].valid()) continue;
                ops[j] = {
                    .buf = bufs[j].data(),
                    .len = aligned_len,
                    .offset = ring_.base_offset() + pending_disk[j].ring_offset,
                    .fd = ring_.read_fd(),
                    .ok = false,
                };
            }
            io_engine_.read_batch(ops.data(), static_cast<uint32_t>(ops.size()));
            for (size_t j = 0; j < pending_disk.size(); ++j) {
                if (!ops[j].ok) continue;
                const auto& e = pending_disk[j];
                DecodedRecord dr = decode_record(bufs[j].bytes());
                if (kRecordHeaderSize + dr.key_len + dr.val_len > e.record_len) continue;
                if (dr.key_len != e.key_len || std::memcmp(dr.key, e.key, dr.key_len) != 0) continue;
                e.result->found = true;
                e.result->actual_len = dr.val_len;
                size_t cpy = std::min(static_cast<size_t>(dr.val_len), e.val_buf_len);
                if (cpy > 0) std::memcpy(e.val_buf, dr.val, cpy);
            }
            pending_disk.clear();
        } else if (should_flush && !io_engine_.uring_enabled()) {
            // Blocking path: we've filled all results
            pending_disk.clear();
            REACT_LOG("blocking_done mark_one_done");
            if (completion) completion->mark_one_done();
        } else {
            REACT_LOG("no_disk mark_one_done");
            if (completion) completion->mark_one_done();
        }
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

    uint64_t abs_offset = static_cast<uint64_t>(file_off) + lr.offset;
    size_t aligned_len = AlignedBuffer::align_up(lr.length);
    auto tmp = AlignedBuffer::allocate(aligned_len);
    ssize_t n = ring_.read(tmp.data(), aligned_len, abs_offset);
    if (n != static_cast<ssize_t>(aligned_len))
        return false;
    std::memcpy(buf.data(), tmp.bytes(), lr.length);
    return true;
}
