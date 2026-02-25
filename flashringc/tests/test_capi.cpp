#include "flashringc/flashringc.h"

#include <cassert>
#include <cstdio>
#include <cstring>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>
#include <chrono>

static const char* TEST_PATH = "/tmp/flashring_capi_test.dat";
static int tests_run = 0;
static int tests_passed = 0;

static void cleanup() {
    for (int i = 0; i < 256; ++i) {
        std::string path = std::string(TEST_PATH) + "." + std::to_string(i);
        ::unlink(path.c_str());
    }
}

#define RUN(name)                                   \
    do {                                            \
        ++tests_run;                                \
        printf("  %-45s ", #name);                  \
        cleanup();                                  \
        name();                                     \
        cleanup();                                  \
        ++tests_passed;                             \
        printf("PASS\n");                           \
    } while (0)

static void test_capi_put_get() {
    FlashRingC* c = flashringc_open(TEST_PATH, 16 * 1024 * 1024, 4);
    assert(c != nullptr);

    const char* key = "hello";
    const char* val = "world";
    int r = flashringc_put(c, key, static_cast<int>(strlen(key)), val, static_cast<int>(strlen(val)), 0);
    assert(r == FLASHRINGC_OK);

    char out[64];
    int out_len = 0;
    r = flashringc_get(c, key, static_cast<int>(strlen(key)), out, sizeof(out), &out_len);
    assert(r == FLASHRINGC_OK);
    assert(out_len == 5);
    assert(std::memcmp(out, "world", 5) == 0);

    flashringc_close(c);
}

static void test_capi_not_found() {
    FlashRingC* c = flashringc_open(TEST_PATH, 16 * 1024 * 1024, 4);
    assert(c != nullptr);

    const char* key = "nonexistent";
    char out[64];
    int out_len = 0;
    int r = flashringc_get(c, key, static_cast<int>(strlen(key)), out, sizeof(out), &out_len);
    assert(r == FLASHRINGC_NOT_FOUND);
    assert(out_len == 0);

    flashringc_close(c);
}

static void test_capi_batch_get() {
    FlashRingC* c = flashringc_open(TEST_PATH, 32 * 1024 * 1024, 4);
    assert(c != nullptr);

    std::vector<std::string> keys(100);
    std::vector<std::string> vals(100);
    for (int i = 0; i < 100; ++i) {
        keys[i] = "bk_" + std::to_string(i);
        vals[i] = "bv_" + std::to_string(i);
        flashringc_put(c, keys[i].data(), static_cast<int>(keys[i].size()),
                       vals[i].data(), static_cast<int>(vals[i].size()), 0);
    }

    std::vector<const char*> key_ptrs(100);
    std::vector<int> key_lens(100);
    for (int i = 0; i < 100; ++i) {
        key_ptrs[i] = keys[i].data();
        key_lens[i] = static_cast<int>(keys[i].size());
    }

    std::vector<char> buf(100 * 64);
    std::vector<char*> val_ptrs(100);
    std::vector<int> val_caps(100, 64);
    std::vector<int> val_lens(100);
    std::vector<int> statuses(100);
    for (int i = 0; i < 100; ++i)
        val_ptrs[i] = buf.data() + i * 64;

    int ok = flashringc_batch_get(c, key_ptrs.data(), key_lens.data(), 100,
                                  val_ptrs.data(), val_caps.data(), val_lens.data(),
                                  statuses.data());
    assert(ok == 100);
    for (int i = 0; i < 100; ++i) {
        assert(statuses[i] == FLASHRINGC_OK);
        assert(val_lens[i] == static_cast<int>(vals[i].size()));
        assert(std::memcmp(val_ptrs[i], vals[i].data(), static_cast<size_t>(val_lens[i])) == 0);
    }

    flashringc_close(c);
}

static void test_capi_batch_mixed() {
    FlashRingC* c = flashringc_open(TEST_PATH, 32 * 1024 * 1024, 4);
    assert(c != nullptr);

    flashringc_put(c, "a", 1, "va", 2, 0);
    flashringc_put(c, "b", 1, "vb", 2, 0);

    const char* keys[] = {"a", "b", "missing"};
    int key_lens[] = {1, 1, 7};
    char out0[8], out1[8], out2[8];
    char* val_ptrs[] = {out0, out1, out2};
    int val_caps[] = {8, 8, 8};
    int val_lens[3];
    int statuses[3];

    int ok = flashringc_batch_get(c, keys, key_lens, 3,
                                  val_ptrs, val_caps, val_lens, statuses);
    assert(ok == 2);
    assert(statuses[0] == FLASHRINGC_OK && val_lens[0] == 2 && std::memcmp(out0, "va", 2) == 0);
    assert(statuses[1] == FLASHRINGC_OK && val_lens[1] == 2 && std::memcmp(out1, "vb", 2) == 0);
    assert(statuses[2] == FLASHRINGC_NOT_FOUND && val_lens[2] == 0);

    flashringc_close(c);
}

static void test_capi_put_get_with_ttl() {
    FlashRingC* c = flashringc_open(TEST_PATH, 16 * 1024 * 1024, 4);
    assert(c != nullptr);

    int r = flashringc_put(c, "k", 1, "v", 1, 0);  // ttl_seconds=0 = no TTL
    assert(r == FLASHRINGC_OK);

    char out[8];
    int out_len = 0;
    r = flashringc_get(c, "k", 1, out, sizeof(out), &out_len);
    assert(r == FLASHRINGC_OK);
    assert(out_len == 1);
    assert(out[0] == 'v');

    flashringc_close(c);
}

static void test_capi_ttl_expiry() {
    FlashRingC* c = flashringc_open(TEST_PATH, 16 * 1024 * 1024, 4);
    assert(c != nullptr);

    int r = flashringc_put(c, "ttl_k", 5, "ttl_v", 5, 1);  // TTL 1 second
    assert(r == FLASHRINGC_OK);

    char out[8];
    int out_len = 0;
    r = flashringc_get(c, "ttl_k", 5, out, sizeof(out), &out_len);
    assert(r == FLASHRINGC_OK);

    std::this_thread::sleep_for(std::chrono::seconds(2));
    out_len = 0;
    r = flashringc_get(c, "ttl_k", 5, out, sizeof(out), &out_len);
    assert(r == FLASHRINGC_NOT_FOUND);
    assert(out_len == 0);

    flashringc_close(c);
}

int main() {
    printf("test_capi\n");
    RUN(test_capi_put_get);
    RUN(test_capi_not_found);
    RUN(test_capi_batch_get);
    RUN(test_capi_batch_mixed);
    RUN(test_capi_put_get_with_ttl);
    RUN(test_capi_ttl_expiry);
    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
