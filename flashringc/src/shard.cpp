#include "shard.h"
#include "aligned_buffer.h"

#include <algorithm>
#include <chrono>
#include <cstring>

Shard::Shard(RingDevice ring, size_t mt_size, uint32_t index_cap)
    : ring_(std::move(ring)),
      memtables_(ring_, mt_size),
      index_(index_cap),
      delete_mgr_(ring_, index_) {}

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
