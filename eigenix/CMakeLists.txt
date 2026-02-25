cmake_minimum_required(VERSION 3.20)
project(eigenix_kmeans LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# ---------- Compile flags ----------
add_compile_options(-O3 -ffast-math)
if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|AMD64")
    add_compile_options(-march=native)
endif()

# ---------- Dependencies ----------
find_package(OpenMP REQUIRED)

# BLAS (OpenBLAS preferred, MKL also works)
set(BLA_VENDOR OpenBLAS)
find_package(BLAS REQUIRED)

# FAISS — try cmake config first, then pkg-config
find_package(faiss QUIET CONFIG)
if(NOT faiss_FOUND)
    find_library(FAISS_LIBRARY NAMES faiss PATHS /usr/local/lib)
    find_path(FAISS_INCLUDE_DIR NAMES faiss/Clustering.h PATHS /usr/local/include)
    if(FAISS_LIBRARY AND FAISS_INCLUDE_DIR)
        add_library(faiss::faiss UNKNOWN IMPORTED)
        set_target_properties(faiss::faiss PROPERTIES
            IMPORTED_LOCATION "${FAISS_LIBRARY}"
            INTERFACE_INCLUDE_DIRECTORIES "${FAISS_INCLUDE_DIR}")
        set(faiss_FOUND TRUE)
    else()
        message(FATAL_ERROR "FAISS not found. Run setup.sh or install FAISS manually.")
    endif()
endif()

# GTest (for tests)
find_package(GTest QUIET)

# ---------- Library: eigenix_kmeans ----------
add_library(eigenix_kmeans STATIC
    src/data_generator.cpp
    src/metrics.cpp
    src/kmeans_blas.cpp
    src/kmeans_faiss.cpp
    src/kmeans_simd.cpp
    src/kmeans_streaming.cpp
)
target_include_directories(eigenix_kmeans PUBLIC include)
target_link_libraries(eigenix_kmeans PUBLIC
    OpenMP::OpenMP_CXX
    ${BLAS_LIBRARIES}
    faiss::faiss
)

# ---------- Benchmark executable ----------
add_executable(eigenix_bench bench/main_bench.cpp)
target_link_libraries(eigenix_bench PRIVATE eigenix_kmeans)

# ---------- Tests ----------
if(GTest_FOUND)
    enable_testing()
    add_executable(eigenix_tests tests/test_correctness.cpp)
    target_link_libraries(eigenix_tests PRIVATE eigenix_kmeans GTest::gtest GTest::gtest_main)
    add_test(NAME eigenix_correctness COMMAND eigenix_tests)
else()
    message(STATUS "GTest not found — tests will not be built")
endif()
