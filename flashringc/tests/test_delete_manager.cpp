#include "flashringc/delete_manager.h"
#include "flashringc/key_index.h"
#include "flashringc/ring_device.h"
#include "flashringc/common.h"

#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <unistd.h>

#define CHECK(cond)                                                         \
    do {                                                                    \
        if (!(cond)) {                                                      \
            fprintf(stderr, "\n  CHECK failed: %s  [%s:%d]\n",             \
                    #cond, __FILE__, __LINE__);                              \
            std::abort();                                                   \
        }                                                                   \
    } while (0)

static const char* TEST_PATH = "/tmp/flashring_dm_test.dat";
static int tests_run = 0;
static int tests_passed = 0;

static void cleanup() { ::unlink(TEST_PATH); }

#define RUN(name)                                       \
    do {                                                \
        ++tests_run;                                    \
        printf("  %-60s ", #name);                      \
        fflush(stdout);                                 \
        cleanup();                                      \
        name();                                         \
        cleanup();                                      \
        ++tests_passed;                                 \
        printf("PASS\n");                               \
    } while (0)

static constexpr uint64_t BLK = 4096;

static void write_blocks(RingDevice& ring, int n) {
    auto buf = AlignedBuffer::allocate(BLK);
    std::memset(buf.data(), 0xAB, BLK);
    for (int i = 0; i < n; i++) {
        int64_t ret = ring.write(buf.data(), BLK);
        CHECK(ret >= 0);
    }
}

static void fill_index_for_memid(KeyIndex& index, uint32_t mem_id, int count) {
    for (int i = 0; i < count; i++) {
        Hash128 h = {static_cast<uint64_t>(mem_id) * 10000ULL + i + 1,
                     0xDEADBEEFULL ^ static_cast<uint64_t>(i)};
        index.put(h, mem_id, static_cast<uint32_t>(i * 64), 64);
    }
}

static int count_entries_for_memid(KeyIndex& index, uint32_t mem_id) {
    int count = 0;
    for (uint32_t i = 0; i < index.capacity(); i++) {
        auto* e = index.entry_at(i);
        if (e && e->mem_id == mem_id) count++;
    }
    return count;
}

// -----------------------------------------------------------------------
// 1. No eviction below threshold
// -----------------------------------------------------------------------
static void test_no_eviction_below_threshold() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.95, .clear_threshold = 0.85};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 16);                       // 80% < 95%
    for (uint32_t m = 0; m < 16; m++)
        fill_index_for_memid(index, m, 5);

    dm.on_put();
    CHECK(!dm.is_evicting());
    CHECK(!dm.is_discarded(0));
    CHECK(!dm.has_backlog());
}

// -----------------------------------------------------------------------
// 2. Eviction triggers at threshold
// -----------------------------------------------------------------------
static void test_eviction_triggers_at_threshold() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.40,
                              .base_k = 4, .max_discards_per_put = 2};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 17);                       // 85% > 80%
    for (uint32_t m = 0; m < 17; m++)
        fill_index_for_memid(index, m, 5);

    dm.on_put();
    CHECK(dm.is_discarded(0));
    CHECK(dm.has_backlog());
    printf("[disc_through=%u] ", dm.discarded_through_mem_id());
}

// -----------------------------------------------------------------------
// 3. Hysteresis stops at clear threshold
// -----------------------------------------------------------------------
static void test_hysteresis_stops_at_clear_threshold() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.40,
                              .base_k = 100, .max_discards_per_put = 20};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 18);                       // 90% > 80%
    for (uint32_t m = 0; m < 18; m++)
        fill_index_for_memid(index, m, 5);

    for (int i = 0; i < 20; i++) dm.on_put();

    CHECK(!dm.is_evicting());
    double util = ring.utilization();
    CHECK(util <= 0.45);
    printf("[util=%.2f] ", util);
}

// -----------------------------------------------------------------------
// 4. Hysteresis prevents thrashing
// -----------------------------------------------------------------------
static void test_hysteresis_no_thrash() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.75, .clear_threshold = 0.40,
                              .base_k = 100, .max_discards_per_put = 1};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 16);                       // 80% > 75%
    for (uint32_t m = 0; m < 16; m++)
        fill_index_for_memid(index, m, 3);

    dm.on_put();
    CHECK(dm.is_evicting());
    CHECK(dm.is_discarded(0));
    CHECK(!dm.is_discarded(1));                   // only 1 block discarded

    dm.on_put();
    CHECK(dm.is_evicting());                      // 70% > 40%

    for (int i = 0; i < 6; i++) dm.on_put();
    CHECK(!dm.is_evicting());                     // should have exited band

    write_blocks(ring, 1);
    dm.on_put();
    CHECK(!dm.is_evicting());
    printf("[no thrash] ");
}

// -----------------------------------------------------------------------
// 5. Index ring triggers discard
// -----------------------------------------------------------------------
static void test_index_ring_triggers_discard() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(20);                           // small index
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.70,
                              .base_k = 100, .max_discards_per_put = 1};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 3);                        // 15% device util (below 80%)
    for (uint32_t m = 0; m < 3; m++)
        fill_index_for_memid(index, m, 6);        // 18/20 = 90% index util (above 80%)

    dm.on_put();
    CHECK(dm.is_discarded(0));
    printf("[disc_through=%u dev=%.2f idx=%.2f] ",
           dm.discarded_through_mem_id(), ring.utilization(), index.utilization());
}

// -----------------------------------------------------------------------
// 6. Amortized cleanup — basic
// -----------------------------------------------------------------------
static void test_amortized_cleanup_basic() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.40,
                              .base_k = 2, .max_discards_per_put = 1};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 17);
    for (uint32_t m = 0; m < 17; m++)
        fill_index_for_memid(index, m, 5);

    int before = count_entries_for_memid(index, 0);
    CHECK(before == 5);

    for (int i = 0; i < 15; i++) dm.on_put();

    int after = count_entries_for_memid(index, 0);
    CHECK(after < before);
    printf("[before=%d after=%d] ", before, after);
}

// -----------------------------------------------------------------------
// 7. Amortized cleanup — amplification
// -----------------------------------------------------------------------
static void test_amortized_cleanup_amplification() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.30,
                              .base_k = 2, .max_discards_per_put = 5};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 18);                       // 90%
    for (uint32_t m = 0; m < 18; m++)
        fill_index_for_memid(index, m, 5);

    dm.on_put();                                  // discards multiple blocks
    CHECK(dm.is_discarded(0));
    CHECK(dm.is_discarded(1));                    // at least 2 blocks

    for (int i = 0; i < 10; i++) dm.on_put();

    int remaining = count_entries_for_memid(index, 0);
    CHECK(remaining == 0);
    printf("[disc_through=%u memid0_remaining=%d] ",
           dm.discarded_through_mem_id(), remaining);
}

// -----------------------------------------------------------------------
// 8. Stale read rejection
// -----------------------------------------------------------------------
static void test_stale_read_rejection() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.40,
                              .base_k = 1, .max_discards_per_put = 1};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 17);
    for (uint32_t m = 0; m < 17; m++)
        fill_index_for_memid(index, m, 5);

    CHECK(!dm.is_discarded(0));
    CHECK(!dm.is_discarded(1));

    dm.on_put();
    CHECK(dm.is_discarded(0));

    uint32_t disc = dm.discarded_through_mem_id();
    CHECK(!dm.is_discarded(disc + 1));
    printf("[disc_through=%u] ", disc);
}

// -----------------------------------------------------------------------
// 9. Multiple rapid discards (capped per put)
// -----------------------------------------------------------------------
static void test_multiple_rapid_discards() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.30,
                              .base_k = 4, .max_discards_per_put = 2};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 19);                       // 95%
    for (uint32_t m = 0; m < 19; m++)
        fill_index_for_memid(index, m, 5);

    dm.on_put();
    CHECK(dm.discarded_through_mem_id() == 1);    // 0-based: 0,1 = 2 blocks

    dm.on_put();
    CHECK(dm.discarded_through_mem_id() == 3);    // +2 more
    printf("[capped OK] ");
}

// -----------------------------------------------------------------------
// 10. Cleanup completes fully
// -----------------------------------------------------------------------
static void test_cleanup_completes() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.40,
                              .base_k = 1, .max_discards_per_put = 1};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 17);
    for (uint32_t m = 0; m < 17; m++)
        fill_index_for_memid(index, m, 5);

    dm.on_put();
    CHECK(dm.has_backlog());

    for (int i = 0; i < 200; i++) {
        dm.on_put();
        dm.on_tick();
    }

    CHECK(!dm.is_evicting());
    CHECK(!dm.has_backlog());
    printf("[complete] ");
}

// -----------------------------------------------------------------------
// 11. Read-only workload — cleanup via on_tick only
// -----------------------------------------------------------------------
static void test_read_only_workload_cleanup() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.40,
                              .base_k = 2, .max_discards_per_put = 1};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 17);
    for (uint32_t m = 0; m < 17; m++)
        fill_index_for_memid(index, m, 3);

    dm.on_put();                                  // triggers discard + partial cleanup
    CHECK(dm.has_backlog());

    int after_put = count_entries_for_memid(index, 0);

    for (int i = 0; i < 20; i++) dm.on_tick();    // only ticks, no puts

    int after_ticks = count_entries_for_memid(index, 0);
    CHECK(after_ticks <= after_put);
    printf("[put=%d tick=%d] ", after_put, after_ticks);
}

// -----------------------------------------------------------------------
// 12. Cleanup continues after eviction band exit
// -----------------------------------------------------------------------
static void test_cleanup_continues_after_eviction_band_exit() {
    auto ring = RingDevice::open(TEST_PATH, 20 * BLK);
    ring.set_memtable_size(BLK);
    KeyIndex index(200);
    DeleteManager::Config cfg{.eviction_threshold = 0.80, .clear_threshold = 0.50,
                              .base_k = 1, .max_discards_per_put = 20};
    DeleteManager dm(cfg, index, ring);

    write_blocks(ring, 17);                       // 85% device util
    for (uint32_t m = 0; m < 17; m++)
        fill_index_for_memid(index, m, 5);        // 85/200 = 42.5% index util

    dm.on_put();                                  // discards until device <= 50%
    CHECK(!dm.is_evicting());
    CHECK(dm.has_backlog());

    for (int i = 0; i < 300; i++) dm.on_put();

    CHECK(!dm.has_backlog());
    printf("[cleanup completed independently] ");
}

// -----------------------------------------------------------------------
// main
// -----------------------------------------------------------------------
int main() {
    setbuf(stdout, nullptr);
    printf("test_delete_manager\n\n");
    auto t0 = std::chrono::steady_clock::now();

    RUN(test_no_eviction_below_threshold);
    RUN(test_eviction_triggers_at_threshold);
    RUN(test_hysteresis_stops_at_clear_threshold);
    RUN(test_hysteresis_no_thrash);
    RUN(test_index_ring_triggers_discard);
    RUN(test_amortized_cleanup_basic);
    RUN(test_amortized_cleanup_amplification);
    RUN(test_stale_read_rejection);
    RUN(test_multiple_rapid_discards);
    RUN(test_cleanup_completes);
    RUN(test_read_only_workload_cleanup);
    RUN(test_cleanup_continues_after_eviction_band_exit);

    auto elapsed = std::chrono::steady_clock::now() - t0;
    double secs = std::chrono::duration<double>(elapsed).count();
    printf("\n%d / %d tests passed (%.1fs total).\n", tests_passed, tests_run, secs);
    return (tests_passed == tests_run) ? 0 : 1;
}
