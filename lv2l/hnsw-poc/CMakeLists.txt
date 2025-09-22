cmake_minimum_required(VERSION 3.14)
project(hnsw_poc CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
add_compile_options(-O3 -march=native)

# If you prefer FetchContent, you can replace this with a fetch step.
# For air-gapped/local, just vendor the repo under external/hnswlib
include_directories(${CMAKE_SOURCE_DIR}/../algomods/hnswlib)


find_package(OpenMP)
if(OpenMP_CXX_FOUND)
  message(STATUS "OpenMP enabled")
  add_compile_definitions(HNSWLIB_USE_OPENMP) # not strictly required; hnswlib uses pragmas
  set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} ${OpenMP_CXX_FLAGS}")
endif()

find_package(Arrow   CONFIG REQUIRED)
find_package(Parquet CONFIG REQUIRED)

# Resolve the correct imported targets across distros
set(ARROW_TGT   "")
set(PARQUET_TGT "")

if (TARGET Arrow::arrow)
  set(ARROW_TGT Arrow::arrow)
elseif (TARGET Arrow::arrow_shared)
  set(ARROW_TGT Arrow::arrow_shared)
elseif (TARGET arrow_shared)
  set(ARROW_TGT arrow_shared)
elseif (TARGET arrow_static)
  set(ARROW_TGT arrow_static)
endif()

if (TARGET Parquet::parquet)
  set(PARQUET_TGT Parquet::parquet)
elseif (TARGET Parquet::parquet_shared)
  set(PARQUET_TGT Parquet::parquet_shared)
elseif (TARGET parquet_shared)
  set(PARQUET_TGT parquet_shared)
elseif (TARGET parquet_static)
  set(PARQUET_TGT parquet_static)
endif()

if (NOT ARROW_TGT OR NOT PARQUET_TGT)
  message(FATAL_ERROR "Could not resolve Arrow/Parquet imported targets on this system.")
endif()

add_executable(hnsw_poc main.cpp)
target_link_libraries(hnsw_poc PRIVATE ${ARROW_TGT} ${PARQUET_TGT})