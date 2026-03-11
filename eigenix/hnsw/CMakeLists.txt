cmake_minimum_required(VERSION 3.12...3.26)

project(hnswlib
    LANGUAGES CXX)

include(GNUInstallDirs)
include(CheckCXXCompilerFlag)

# --- Header-only interface library (unchanged hnswlib core) ---
add_library(hnswlib INTERFACE)
add_library(hnswlib::hnswlib ALIAS hnswlib)

target_include_directories(hnswlib INTERFACE
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}>
    $<INSTALL_INTERFACE:${CMAKE_INSTALL_INCLUDEDIR}>)

# Install
install(DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/hnswlib
    DESTINATION ${CMAKE_INSTALL_INCLUDEDIR})

install(TARGETS hnswlib
    EXPORT hnswlibTargets)

install(EXPORT hnswlibTargets
    FILE hnswlibConfig.cmake
    NAMESPACE hnswlib::
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/hnswlib)

# --- Collection static library (C++20, dual-segment architecture) ---
option(HNSWLIB_COLLECTION "Build the Collection library (requires C++20)" ON)

if(HNSWLIB_COLLECTION)
    add_library(hnswlib_collection STATIC
        hnswlib/hnsw_wrapper.cpp
        hnswlib/collection.cpp
        hnswlib/rebuilder.cpp
    )

    target_include_directories(hnswlib_collection PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}>
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/hnswlib>
    )

    set_target_properties(hnswlib_collection PROPERTIES
        CXX_STANDARD 20
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
    )

    target_link_libraries(hnswlib_collection PUBLIC hnswlib)

    find_package(Threads REQUIRED)
    target_link_libraries(hnswlib_collection PUBLIC Threads::Threads)

    # SIMD flags (x86_64 only)
    if(CMAKE_SYSTEM_PROCESSOR MATCHES "(x86_64|AMD64|amd64)")
        check_cxx_compiler_flag("-msse4.2" HAS_SSE42)
        check_cxx_compiler_flag("-mavx" HAS_AVX)
        if(HAS_SSE42)
            target_compile_options(hnswlib_collection PRIVATE -msse4.2)
        endif()
        if(HAS_AVX)
            target_compile_options(hnswlib_collection PRIVATE -mavx)
        endif()
    endif()

    target_compile_options(hnswlib_collection PRIVATE -O2 -Wall -Wextra -Wpedantic)

    install(TARGETS hnswlib_collection
        ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR})
endif()

# --- Collection tests (Google Test) ---
option(HNSWLIB_COLLECTION_TESTS "Build Collection tests" OFF)

if(HNSWLIB_COLLECTION_TESTS AND HNSWLIB_COLLECTION)
    find_package(GTest REQUIRED)

    add_executable(collection_test tests/collection_test.cpp)
    target_link_libraries(collection_test PRIVATE hnswlib_collection GTest::gtest GTest::gtest_main)
    set_target_properties(collection_test PROPERTIES
        CXX_STANDARD 20
        CXX_STANDARD_REQUIRED ON
    )

    # ASAN target (for use-after-free detection)
    add_executable(collection_test_asan tests/collection_test.cpp)
    target_link_libraries(collection_test_asan PRIVATE hnswlib_collection GTest::gtest GTest::gtest_main)
    set_target_properties(collection_test_asan PROPERTIES
        CXX_STANDARD 20
        CXX_STANDARD_REQUIRED ON
    )
    target_compile_options(collection_test_asan PRIVATE -fsanitize=address -fno-omit-frame-pointer)
    target_link_options(collection_test_asan PRIVATE -fsanitize=address)

    # TSAN target (for data race detection)
    add_executable(collection_test_tsan tests/collection_test.cpp)
    target_link_libraries(collection_test_tsan PRIVATE hnswlib_collection GTest::gtest GTest::gtest_main)
    set_target_properties(collection_test_tsan PROPERTIES
        CXX_STANDARD 20
        CXX_STANDARD_REQUIRED ON
    )
    target_compile_options(collection_test_tsan PRIVATE -fsanitize=thread)
    target_link_options(collection_test_tsan PRIVATE -fsanitize=thread)
endif()

# --- Collection benchmarks (Google Benchmark) ---
option(HNSWLIB_COLLECTION_BENCHMARKS "Build Collection benchmarks" OFF)

if(HNSWLIB_COLLECTION_BENCHMARKS AND HNSWLIB_COLLECTION)
    find_package(benchmark REQUIRED)

    add_executable(collection_benchmark bench/benchmark.cpp)
    target_link_libraries(collection_benchmark PRIVATE hnswlib_collection benchmark::benchmark)
    set_target_properties(collection_benchmark PROPERTIES
        CXX_STANDARD 20
        CXX_STANDARD_REQUIRED ON
    )
    target_compile_options(collection_benchmark PRIVATE -O2 -DNDEBUG)
endif()

# --- Upstream C++ tests (hnswlib core, C++11) ---
option(HNSWLIB_UPSTREAM_TESTS "Build upstream hnswlib C++ tests" OFF)

if(HNSWLIB_UPSTREAM_TESTS)
    set(CMAKE_CXX_STANDARD 11)

    add_executable(multivector_search_test tests/multivector_search_test.cpp)
    target_link_libraries(multivector_search_test hnswlib)

    add_executable(epsilon_search_test tests/epsilon_search_test.cpp)
    target_link_libraries(epsilon_search_test hnswlib)

    add_executable(test_updates tests/updates_test.cpp)
    target_link_libraries(test_updates hnswlib)

    add_executable(searchKnnCloserFirst_test tests/searchKnnCloserFirst_test.cpp)
    target_link_libraries(searchKnnCloserFirst_test hnswlib)

    add_executable(searchKnnWithFilter_test tests/searchKnnWithFilter_test.cpp)
    target_link_libraries(searchKnnWithFilter_test hnswlib)

    add_executable(multiThreadLoad_test tests/multiThreadLoad_test.cpp)
    target_link_libraries(multiThreadLoad_test hnswlib)

    add_executable(multiThread_replace_test tests/multiThread_replace_test.cpp)
    target_link_libraries(multiThread_replace_test hnswlib)

    add_executable(hnsw_main tests/main.cpp tests/sift_1b.cpp)
    target_link_libraries(hnsw_main hnswlib)
endif()
