cmake_minimum_required(VERSION 3.14)
project(flashringc CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

include(FetchContent)

# --- abseil-cpp ---
FetchContent_Declare(
    abseil-cpp
    GIT_REPOSITORY https://github.com/abseil/abseil-cpp.git
    GIT_TAG        20240722.0
    GIT_SHALLOW    TRUE
)
set(ABSL_PROPAGATE_CXX_STD ON)
FetchContent_MakeAvailable(abseil-cpp)

# --- xxHash (header-only, used via XXH_INLINE_ALL) ---
FetchContent_Declare(
    xxHash
    GIT_REPOSITORY https://github.com/Cyan4973/xxHash.git
    GIT_TAG        v0.8.2
    GIT_SHALLOW    TRUE
)
FetchContent_Populate(xxHash)
set(XXHASH_INCLUDE_DIR "${xxhash_SOURCE_DIR}")

# --- liburing (Linux only) ---
if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    find_library(URING_LIB uring)
    if(URING_LIB)
        set(HAVE_IO_URING TRUE)
        message(STATUS "Found liburing: ${URING_LIB}")
    else()
        set(HAVE_IO_URING FALSE)
        message(STATUS "liburing not found — reactor will use pread fallback")
    endif()
else()
    set(HAVE_IO_URING FALSE)
    message(STATUS "Non-Linux platform — reactor uses pread fallback")
endif()

# --- Library ---
add_library(flashring
    src/ring_device.cpp
    src/memtable_manager.cpp
    src/key_index.cpp
    src/delete_manager.cpp
    src/semaphore_pool.cpp
    src/shard_reactor.cpp
    src/cache.cpp
    src/flashringc_capi.cpp
)

target_include_directories(flashring
    PUBLIC  include/
    PRIVATE ${XXHASH_INCLUDE_DIR}
)

target_link_libraries(flashring
    PUBLIC  absl::flat_hash_map absl::crc32c
    PRIVATE pthread
)

if(HAVE_IO_URING)
    target_compile_definitions(flashring PUBLIC HAVE_IO_URING)
    target_link_libraries(flashring PRIVATE ${URING_LIB})
endif()

# --- Tests ---
add_executable(test_semaphore_pool tests/test_semaphore_pool.cpp)
target_link_libraries(test_semaphore_pool PRIVATE flashring)

add_executable(test_mpsc_queue tests/test_mpsc_queue.cpp)
target_link_libraries(test_mpsc_queue PRIVATE flashring)

add_executable(test_shard_reactor tests/test_shard_reactor.cpp)
target_link_libraries(test_shard_reactor PRIVATE flashring)

add_executable(test_cache tests/test_cache.cpp)
target_link_libraries(test_cache PRIVATE flashring)

add_executable(bench_cache tests/bench_cache.cpp)
target_link_libraries(bench_cache PRIVATE flashring)

add_executable(bench_heavy tests/bench_heavy.cpp)
target_link_libraries(bench_heavy PRIVATE flashring)

add_executable(bench_long_run tests/bench_long_run.cpp)
target_link_libraries(bench_long_run PRIVATE flashring)

add_executable(test_eviction tests/test_eviction.cpp)
target_link_libraries(test_eviction PRIVATE flashring)

add_executable(test_delete_manager tests/test_delete_manager.cpp)
target_link_libraries(test_delete_manager PRIVATE flashring)

add_executable(test_capi tests/test_capi.cpp)
target_link_libraries(test_capi PRIVATE flashring)

# --- Legacy tests (kept for reference, may need updating) ---
add_executable(ring_test tests/ring_test.cpp)
target_link_libraries(ring_test PRIVATE flashring)

add_executable(index_test tests/index_test.cpp)
target_link_libraries(index_test PRIVATE flashring)

add_executable(index_perf_test tests/index_perf_test.cpp)
target_link_libraries(index_perf_test PRIVATE flashring)

add_executable(perf_test tests/perf_test.cpp)
target_link_libraries(perf_test PRIVATE flashring)
