add_executable(wal_client wal_client.cc)
target_link_libraries(wal_client PRIVATE mwal)
target_include_directories(wal_client PRIVATE
  ${CMAKE_SOURCE_DIR}/src
  ${CMAKE_SOURCE_DIR}/include
)
