#include "flashringc/ring_device.h"
#include "flashringc/memtable_manager.h"

#include <cassert>
#include <cstdio>
#include <cstring>
#include <iostream>
#include <unistd.h>

static const char* TEST_FILE = "/tmp/flashring_test.dat";
constexpr uint64_t RING_SIZE    = 1ULL * 1024 * 1024;  // 1 MB
constexpr size_t   MT_SIZE      = 64 * 1024;            // 64 KB

// ── helpers ──────────────────────────────────────────────────────────────────

static void cleanup() { ::unlink(TEST_FILE); }

#define ASSERT_EQ(a, b)                                                    \
    do {                                                                   \
        if ((a) != (b)) {                                                  \
            std::cerr << __FILE__ << ":" << __LINE__ << "  ASSERT_EQ("     \
                      << #a << ", " << #b << ")  got " << (a) << " vs "    \
                      << (b) << "\n";                                      \
            std::abort();                                                  \
        }                                                                  \
    } while (0)

// ── test: basic write / read ─────────────────────────────────────────────────

static void test_basic_write_read() {
    cleanup();
    auto ring = RingDevice::open(TEST_FILE, RING_SIZE);

    ASSERT_EQ(ring.capacity(), RING_SIZE);
    ASSERT_EQ(ring.write_offset(), 0ULL);
    assert(!ring.wrapped());

    auto wbuf = AlignedBuffer::allocate(kBlockSize);
    std::memset(wbuf.data(), 0xAB, kBlockSize);

    int64_t off = ring.write(wbuf.data(), kBlockSize);
    ASSERT_EQ(off, 0LL);
    ASSERT_EQ(ring.write_offset(), (uint64_t)kBlockSize);

    auto rbuf = AlignedBuffer::allocate(kBlockSize);
    ssize_t n = ring.read(rbuf.data(), kBlockSize, 0);
    ASSERT_EQ(n, (ssize_t)kBlockSize);
    assert(std::memcmp(wbuf.data(), rbuf.data(), kBlockSize) == 0);

    std::cout << "  test_basic_write_read:  PASS\n";
    cleanup();
}

// ── test: wrap-around ────────────────────────────────────────────────────────

static void test_wrap_around() {
    cleanup();
    auto ring = RingDevice::open(TEST_FILE, RING_SIZE);

    auto buf = AlignedBuffer::allocate(kBlockSize);
    int writes = 0;

    while (!ring.wrapped()) {
        std::memset(buf.data(), static_cast<int>(writes & 0xFF), kBlockSize);
        int64_t off = ring.write(buf.data(), kBlockSize);
        assert(off >= 0);
        ++writes;
    }

    // After wrap the write offset is back near 0.
    assert(ring.write_offset() <= kBlockSize);

    // Write one more block after wrap and read it back.
    uint8_t marker = 0xCD;
    std::memset(buf.data(), marker, kBlockSize);
    int64_t post_wrap_off = ring.write(buf.data(), kBlockSize);
    assert(post_wrap_off >= 0);

    auto rbuf = AlignedBuffer::allocate(kBlockSize);
    ssize_t n = ring.read(rbuf.data(), kBlockSize,
                          static_cast<uint64_t>(post_wrap_off));
    ASSERT_EQ(n, (ssize_t)kBlockSize);
    ASSERT_EQ(rbuf.bytes()[0], marker);

    std::cout << "  test_wrap_around:       PASS  (" << writes
              << " writes before wrap)\n";
    cleanup();
}

// ── test: read_unaligned ─────────────────────────────────────────────────────

static void test_read_unaligned() {
    cleanup();
    auto ring = RingDevice::open(TEST_FILE, RING_SIZE);

    // Write a block filled with a known pattern.
    auto wbuf = AlignedBuffer::allocate(kBlockSize);
    for (size_t i = 0; i < kBlockSize; ++i)
        wbuf.bytes()[i] = static_cast<uint8_t>(i & 0xFF);

    int64_t off = ring.write(wbuf.data(), kBlockSize);
    assert(off == 0);

    // Read 10 bytes starting at offset 100 (neither aligned).
    uint8_t tmp[10] = {};
    ssize_t n = ring.read_unaligned(tmp, 10, 100);
    ASSERT_EQ(n, (ssize_t)10);
    for (int i = 0; i < 10; ++i)
        ASSERT_EQ(tmp[i], static_cast<uint8_t>((100 + i) & 0xFF));

    std::cout << "  test_read_unaligned:    PASS\n";
    cleanup();
}

// ── test: memtable put + in-memory read ──────────────────────────────────────

static void test_memtable_put_read() {
    cleanup();
    auto ring = RingDevice::open(TEST_FILE, RING_SIZE);
    MemtableManager mgr(ring, MT_SIZE);

    const char* payload = "hello flashring cpp";
    size_t plen = std::strlen(payload);

    auto wr = mgr.put(payload, plen);
    ASSERT_EQ(wr.length, (uint16_t)plen);

    char readback[64] = {};
    bool found = mgr.try_read_from_memory(readback, wr.length,
                                          wr.mem_id, wr.offset);
    assert(found);
    assert(std::memcmp(readback, payload, plen) == 0);

    std::cout << "  test_memtable_put_read: PASS\n";
    cleanup();
}

// ── test: memtable swap + disk read ──────────────────────────────────────────

static void test_memtable_swap() {
    cleanup();
    auto ring = RingDevice::open(TEST_FILE, RING_SIZE);
    MemtableManager mgr(ring, MT_SIZE);

    // Write a known record.
    const char* payload = "persist me";
    size_t plen = std::strlen(payload);
    auto first = mgr.put(payload, plen);

    // Fill the active memtable until it needs flush, then swap + sync flush.
    char filler[1024];
    std::memset(filler, 'X', sizeof(filler));
    while (!mgr.needs_flush(sizeof(filler))) {
        mgr.put(filler, sizeof(filler));
    }

    // Synchronously flush (swap + write to disk + complete).
    mgr.flush_sync();

    // The first record should now be on disk.
    int64_t file_off = mgr.file_offset_for(first.mem_id);
    assert(file_off >= 0);

    // Read it back via the ring device.
    char diskbuf[64] = {};
    ssize_t n = ring.read_unaligned(diskbuf, first.length,
                                    static_cast<uint64_t>(file_off) + first.offset);
    ASSERT_EQ(n, (ssize_t)first.length);
    assert(std::memcmp(diskbuf, payload, plen) == 0);

    std::cout << "  test_memtable_swap:     PASS\n";
    cleanup();
}

// ── test: discard (no-op on file, should not error) ──────────────────────────

static void test_discard_noop() {
    cleanup();
    auto ring = RingDevice::open(TEST_FILE, RING_SIZE);

    auto buf = AlignedBuffer::allocate(kBlockSize);
    std::memset(buf.data(), 0xFF, kBlockSize);
    ring.write(buf.data(), kBlockSize);

    int rc = ring.discard(0, kBlockSize);
    ASSERT_EQ(rc, 0);

    std::cout << "  test_discard_noop:      PASS\n";
    cleanup();
}

// ── main ─────────────────────────────────────────────────────────────────────

int main() {
    std::cout << "flashringc tests\n";
    std::cout << "─────────────────────────────────\n";

    test_basic_write_read();
    test_wrap_around();
    test_read_unaligned();
    test_memtable_put_read();
    test_memtable_swap();
    test_discard_noop();

    std::cout << "─────────────────────────────────\n";
    std::cout << "all tests passed.\n";
    return 0;
}
