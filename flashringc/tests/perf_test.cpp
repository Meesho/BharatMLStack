#include "ring_device.h"
#include "aligned_buffer.h"

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <numeric>
#include <random>
#include <vector>
#include <unistd.h>
#include <sys/stat.h>

// Default fallback for file-backed mode (macOS dev, CI, etc.).
static constexpr const char* DEFAULT_PATH = "/tmp/flashring_bench.dat";
constexpr uint64_t FILE_CAPACITY = 2ULL * 1024 * 1024 * 1024; // 2 GB (files only)
constexpr uint64_t WARMUP_BYTES  = 2ULL * 1024 * 1024 * 1024; // 2 GB

struct Payload {
    size_t size;
    const char* label;
    int iters;
};

static const Payload kPayloads[] = {
    {1 << 10,   "1K", 10000},
    {1 << 11,   "2K", 10000},
    {1 << 12,   "4K", 10000},
    {1 << 13,   "8K",  5000},
    {1 << 14,  "16K",  5000},
    {1 << 15,  "32K",  5000},
    {1 << 16,  "64K",  5000},
    {1 << 17, "128K",  1000},
    {1 << 18, "256K",  1000},
    {1 << 19, "512K",  1000},
    {1 << 20,   "1M",  1000},
    {1 << 21,   "2M",   200},
    {1 << 22,   "4M",   200},
    {1 << 23,   "8M",   100},
    {1 << 24,  "16M",   100},
};

using Clock = std::chrono::high_resolution_clock;
using us_d  = std::chrono::duration<double, std::micro>;

struct Stats {
    double avg_us, p50_us, p99_us, mbps;
};

static Stats summarise(std::vector<double>& v, size_t io_bytes) {
    std::sort(v.begin(), v.end());
    size_t n = v.size();
    double total_us = std::accumulate(v.begin(), v.end(), 0.0);
    return {
        total_us / static_cast<double>(n),
        v[n / 2],
        v[std::min(n - 1, static_cast<size_t>(n * 0.99))],
        static_cast<double>(io_bytes) * static_cast<double>(n)
            / (total_us / 1e6) / (1024.0 * 1024.0),
    };
}

static bool is_block_device(const char* path) {
    struct stat st{};
    return (::stat(path, &st) == 0 && S_ISBLK(st.st_mode));
}

// Warmup: write WARMUP_BYTES so SSD FTL / filesystem blocks are initialised.
static void warmup(const char* path, bool is_blk) {
    uint64_t cap = is_blk ? 0 : FILE_CAPACITY;
    auto ring = RingDevice::open(path, cap);

    uint64_t warmup_limit = std::min(WARMUP_BYTES, ring.capacity());
    std::printf("warmup: writing %llu MB ...",
                static_cast<unsigned long long>(warmup_limit >> 20));
    std::fflush(stdout);

    constexpr size_t CHUNK = 1 << 20;
    auto buf = AlignedBuffer::allocate(CHUNK);
    std::memset(buf.data(), 0, CHUNK);

    for (uint64_t written = 0; written + CHUNK <= warmup_limit; written += CHUNK)
        ring.write(buf.data(), CHUNK);

    std::printf(" done.\n\n");
}

int main(int argc, char* argv[]) {
    const char* path = (argc > 1) ? argv[1] : DEFAULT_PATH;
    bool is_blk = is_block_device(path);

    if (!is_blk) ::unlink(path);

    const char* mode = is_blk ? "raw block device" : "file-backed";
    std::printf("flashringc perf benchmark  (O_DIRECT, %s)\n", mode);
    std::printf("path          : %s\n", path);

    // Open once to print capacity, then close.
    {
        uint64_t cap_arg = is_blk ? 0 : FILE_CAPACITY;
        auto probe = RingDevice::open(path, cap_arg);
        std::printf("ring capacity : %llu MB\n\n",
                    static_cast<unsigned long long>(probe.capacity() >> 20));
    }

    warmup(path, is_blk);

    // header
    std::printf(
        "%-7s %8s %6s | %9s %9s %9s %9s | %9s %9s %9s %9s\n",
        "payload", "io_size", "iters",
        "wr_avg", "wr_p50", "wr_p99", "wr_MB/s",
        "rd_avg", "rd_p50", "rd_p99", "rd_MB/s");
    std::printf(
        "%-7s %8s %6s | %9s %9s %9s %9s | %9s %9s %9s %9s\n",
        "", "(bytes)", "",
        "(us)", "(us)", "(us)", "",
        "(us)", "(us)", "(us)", "");
    std::printf(
        "------------------------+------------------------------------------"
        "+------------------------------------------\n");

    std::mt19937 rng(42);

    for (const auto& p : kPayloads) {
        const size_t io_sz = AlignedBuffer::align_up(p.size);
        uint64_t cap_arg = is_blk ? 0 : FILE_CAPACITY;
        auto ring = RingDevice::open(path, cap_arg);

        // Clamp iterations so total data fits in usable region.
        uint64_t usable = std::min(ring.capacity(), WARMUP_BYTES);
        int iters = p.iters;
        if (static_cast<uint64_t>(io_sz) * iters > usable)
            iters = static_cast<int>(usable / io_sz) - 1;

        auto wbuf = AlignedBuffer::allocate(io_sz);
        auto rbuf = AlignedBuffer::allocate(io_sz);
        for (size_t i = 0; i < io_sz; ++i)
            wbuf.bytes()[i] = static_cast<uint8_t>((i * 7 + 13) & 0xFF);

        // ── WRITE (sequential) ───────────────────────────────────────────
        std::vector<double>  wr_lat(iters);
        std::vector<int64_t> offsets(iters);

        for (int i = 0; i < iters; ++i) {
            auto t0    = Clock::now();
            offsets[i] = ring.write(wbuf.data(), io_sz);
            auto t1    = Clock::now();
            wr_lat[i]  = std::chrono::duration_cast<us_d>(t1 - t0).count();
        }
        auto ws = summarise(wr_lat, io_sz);

        // ── READ (random order to defeat prefetch / readahead) ───────────
        std::vector<int> idx(iters);
        std::iota(idx.begin(), idx.end(), 0);
        std::shuffle(idx.begin(), idx.end(), rng);

        std::vector<double> rd_lat(iters);

        for (int i = 0; i < iters; ++i) {
            auto t0   = Clock::now();
            ring.read(rbuf.data(), io_sz,
                      static_cast<uint64_t>(offsets[idx[i]]));
            auto t1   = Clock::now();
            rd_lat[i] = std::chrono::duration_cast<us_d>(t1 - t0).count();
        }
        auto rs = summarise(rd_lat, io_sz);

        std::printf(
            "%-7s %8zu %6d | %9.1f %9.1f %9.1f %9.1f | %9.1f %9.1f %9.1f %9.1f\n",
            p.label, io_sz, iters,
            ws.avg_us, ws.p50_us, ws.p99_us, ws.mbps,
            rs.avg_us, rs.p50_us, rs.p99_us, rs.mbps);
    }

    if (!is_blk) ::unlink(path);
    std::printf("\ndone.\n");
    return 0;
}
