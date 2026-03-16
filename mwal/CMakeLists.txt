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
  # Propagate link directory so executables that link mwal can find libzstd
  if(ZSTD_LIBRARY_DIRS)
    target_link_directories(mwal PUBLIC ${ZSTD_LIBRARY_DIRS})
  elseif(ZSTD_LIBRARY)
    get_filename_component(_zstd_dir "${ZSTD_LIBRARY}" DIRECTORY)
    target_link_directories(mwal PUBLIC "${_zstd_dir}")
  endif()
  target_link_libraries(mwal PUBLIC ${ZSTD_LIBRARIES})
else()
  message(STATUS "zstd not found — WAL compression disabled. Install with: brew install zstd")
endif()

option(MWAL_BUILD_TESTS "Build mwal unit tests" ON)
option(MWAL_BUILD_BENCH "Build mwal benchmarks" ON)
option(MWAL_BUILD_EXAMPLES "Build mwal example client" ON)
option(MWAL_BUILD_REPLICATION "Build mwal replication layer (requires gRPC + NuRaft)" ON)

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

# ── Replication layer (gRPC + NuRaft) ────────────────────────────────────
if(MWAL_BUILD_REPLICATION)
  # ── gRPC + Protobuf (system-installed) ──
  find_package(Protobuf REQUIRED CONFIG)
  find_package(gRPC REQUIRED CONFIG)

  find_program(PROTOC_EXECUTABLE protoc REQUIRED)
  find_program(GRPC_CPP_PLUGIN grpc_cpp_plugin REQUIRED)

  message(STATUS "Using system protoc: ${PROTOC_EXECUTABLE}")
  message(STATUS "Using system grpc_cpp_plugin: ${GRPC_CPP_PLUGIN}")

  # ── NuRaft ──
  set(FETCHCONTENT_QUIET OFF)
  FetchContent_Declare(
    nuraft
    GIT_REPOSITORY https://github.com/eBay/NuRaft.git
    GIT_TAG        v3.0.0
    GIT_SHALLOW    TRUE
  )
  set(DISABLE_SSL ON CACHE BOOL "" FORCE)
  set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)
  FetchContent_MakeAvailable(nuraft)

  # ── Proto generation ──
  set(PROTO_SRC_DIR ${CMAKE_CURRENT_SOURCE_DIR}/proto)
  set(PROTO_GEN_DIR ${CMAKE_CURRENT_BINARY_DIR}/proto_gen)
  file(MAKE_DIRECTORY ${PROTO_GEN_DIR})

  set(REPL_PROTO ${PROTO_SRC_DIR}/replication.proto)
  set(REPL_PROTO_SRCS ${PROTO_GEN_DIR}/replication.pb.cc)
  set(REPL_PROTO_HDRS ${PROTO_GEN_DIR}/replication.pb.h)
  set(REPL_GRPC_SRCS  ${PROTO_GEN_DIR}/replication.grpc.pb.cc)
  set(REPL_GRPC_HDRS  ${PROTO_GEN_DIR}/replication.grpc.pb.h)

  add_custom_command(
    OUTPUT ${REPL_PROTO_SRCS} ${REPL_PROTO_HDRS} ${REPL_GRPC_SRCS} ${REPL_GRPC_HDRS}
    COMMAND ${PROTOC_EXECUTABLE}
      --proto_path=${PROTO_SRC_DIR}
      --cpp_out=${PROTO_GEN_DIR}
      --grpc_out=${PROTO_GEN_DIR}
      --plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN}
      ${REPL_PROTO}
    DEPENDS ${REPL_PROTO}
    COMMENT "Generating gRPC/protobuf sources for replication.proto"
  )

  # ── mwal_replication library ──
  set(REPL_SOURCES
    src/replication/replication_manager.cc
    src/replication/isr_tracker.cc
    src/replication/replication_service.cc
    src/replication/replication_client.cc
    src/replication/raft_state_machine.cc
    src/replication/raft_log_store.cc
    ${REPL_PROTO_SRCS}
    ${REPL_GRPC_SRCS}
  )

  add_library(mwal_replication STATIC ${REPL_SOURCES})

  target_include_directories(mwal_replication PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
    ${PROTO_GEN_DIR}
  )
  target_link_libraries(mwal_replication PUBLIC
    mwal
    gRPC::grpc++
    gRPC::grpc++_reflection
    protobuf::libprotobuf
    NuRaft::static_lib
  )

  # ── Replication tests ──
  if(MWAL_BUILD_TESTS)
    function(mwal_add_repl_test name)
      add_executable(${name} test/${name}.cc)
      target_link_libraries(${name} PRIVATE mwal_replication GTest::gtest GTest::gtest_main)
      target_include_directories(${name} PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src
        ${CMAKE_CURRENT_SOURCE_DIR}/include
        ${PROTO_GEN_DIR}
      )
      add_test(NAME ${name} COMMAND ${name})
    endfunction()

    mwal_add_repl_test(isr_tracker_test)
    mwal_add_repl_test(replication_manager_test)
    mwal_add_repl_test(replication_e2e_test)
  endif()
endif()

if(MWAL_BUILD_EXAMPLES)
  add_subdirectory(examples)
endif()
