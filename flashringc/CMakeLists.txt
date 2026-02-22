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

add_library(flashring
    src/ring_device.cpp
    src/memtable.cpp
    src/key_index.cpp
)
target_include_directories(flashring PUBLIC src/ PRIVATE ${XXHASH_INCLUDE_DIR})
target_link_libraries(flashring
    PUBLIC  absl::flat_hash_map
    PRIVATE pthread
)

add_executable(ring_test tests/ring_test.cpp)
target_link_libraries(ring_test PRIVATE flashring)

add_executable(perf_test tests/perf_test.cpp)
target_link_libraries(perf_test PRIVATE flashring)

add_executable(index_test tests/index_test.cpp)
target_link_libraries(index_test PRIVATE flashring)

add_executable(index_perf_test tests/index_perf_test.cpp)
target_link_libraries(index_perf_test PRIVATE flashring)
