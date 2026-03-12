# Distributed K-Means POC — coordinator + worker executables.
# No new external dependencies beyond what eigenix_kmeans already provides.

add_executable(dist_coordinator src/dist_coordinator.cpp src/dist_net.cpp)
target_include_directories(dist_coordinator PRIVATE
    include
    ${CMAKE_SOURCE_DIR}/include
)
target_link_libraries(dist_coordinator PRIVATE eigenix_kmeans)

add_executable(dist_worker src/dist_worker.cpp src/dist_net.cpp)
target_include_directories(dist_worker PRIVATE
    include
    ${CMAKE_SOURCE_DIR}/include
)
target_link_libraries(dist_worker PRIVATE eigenix_kmeans)
