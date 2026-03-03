cmake_minimum_required(VERSION 3.14)
project(mwal LANGUAGES CXX)

set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -Wextra -Wshadow -Woverloaded-virtual -Wnon-virtual-dtor -pthread")
set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -O2 -fno-omit-frame-pointer")
set(CMAKE_CXX_FLAGS_DEBUG "${CMAKE_CXX_FLAGS_DEBUG} -g -O0")

# ThreadSanitizer support for race condition detection
option(MWAL_ENABLE_TSAN "Enable ThreadSanitizer for race condition detection" OFF)
if(MWAL_ENABLE_TSAN)
  set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fsanitize=thread -fno-omit-frame-pointer")
  set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -fsanitize=thread")
  message(STATUS "ThreadSanitizer enabled for WAL concurrency testing")
endif()

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

include(FetchContent)

set(MWAL_SOURCES
  src/slice.cc
  src/status.cc
  src/util/coding.cc
  src/util/crc32c.cc
  src/env/env_posix.cc
  src/file/writable_file_writer.cc
  src/file/sequential_file_reader.cc
  src/log/log_writer.cc
  src/log/log_reader.cc
  src/wal/write_batch.cc
  src/wal/wal_edit.cc
  src/wal/wal_manager.cc
  src/wal/write_thread.cc
  src/wal/write_coalescer.cc
  src/wal/wal_compressor.cc
  src/wal/db_wal.cc
  src/wal/wal_iterator.cc
)

add_library(mwal STATIC ${MWAL_SOURCES})

target_include_directories(mwal PUBLIC
  ${CMAKE_CURRENT_SOURCE_DIR}/include
)
target_include_directories(mwal PRIVATE
  ${CMAKE_CURRENT_SOURCE_DIR}/src
)

# Optional zstd compression support
find_package(PkgConfig QUIET)
if(PkgConfig_FOUND)
  pkg_check_modules(ZSTD QUIET libzstd)
endif()
if(NOT ZSTD_FOUND)
  find_library(ZSTD_LIBRARY NAMES zstd)
  find_path(ZSTD_INCLUDE_DIR NAMES zstd.h)
  if(ZSTD_LIBRARY AND ZSTD_INCLUDE_DIR)
    set(ZSTD_FOUND TRUE)
    set(ZSTD_LIBRARIES ${ZSTD_LIBRARY})
    set(ZSTD_INCLUDE_DIRS ${ZSTD_INCLUDE_DIR})
  endif()
endif()

if(ZSTD_FOUND)
  message(STATUS "zstd found — WAL compression enabled")
  target_compile_definitions(mwal PUBLIC MWAL_HAVE_ZSTD=1)
  target_include_directories(mwal PRIVATE ${ZSTD_INCLUDE_DIRS})
  target_link_libraries(mwal PUBLIC ${ZSTD_LIBRARIES})
else()
  message(STATUS "zstd not found — WAL compression disabled. Install with: brew install zstd")
endif()

option(MWAL_BUILD_TESTS "Build mwal unit tests" ON)
option(MWAL_BUILD_BENCH "Build mwal benchmarks" ON)
option(MWAL_BUILD_EXAMPLES "Build mwal example client" ON)

if(MWAL_BUILD_TESTS)
  FetchContent_Declare(
    googletest
    GIT_REPOSITORY https://github.com/google/googletest.git
    GIT_TAG v1.14.0
  )
  set(gtest_force_shared_crt ON CACHE BOOL "" FORCE)
  FetchContent_MakeAvailable(googletest)

  enable_testing()
  add_subdirectory(test)
endif()

if(MWAL_BUILD_BENCH)
  find_package(benchmark QUIET)
  if(benchmark_FOUND)
    add_subdirectory(bench)
  else()
    message(STATUS "Google Benchmark not found, skipping benchmarks. Install with: brew install google-benchmark")
  endif()
endif()

if(MWAL_BUILD_EXAMPLES)
  add_subdirectory(examples)
endif()
