cmake_minimum_required(VERSION 3.14)
project(flashringc CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

add_library(flashring
    src/ring_device.cpp
    src/memtable.cpp
)
target_include_directories(flashring PUBLIC src/)
target_link_libraries(flashring PRIVATE pthread)

add_executable(ring_test tests/ring_test.cpp)
target_link_libraries(ring_test PRIVATE flashring)

add_executable(perf_test tests/perf_test.cpp)
target_link_libraries(perf_test PRIVATE flashring)
