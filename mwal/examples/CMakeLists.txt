add_executable(wal_client wal_client.cc)
target_link_libraries(wal_client PRIVATE mwal)
target_include_directories(wal_client PRIVATE
  ${CMAKE_SOURCE_DIR}/src
  ${CMAKE_SOURCE_DIR}/include
)

add_executable(wal_dump wal_dump.cc)
target_link_libraries(wal_dump PRIVATE mwal)
target_include_directories(wal_dump PRIVATE
  ${CMAKE_SOURCE_DIR}/src
  ${CMAKE_SOURCE_DIR}/include
)

if(TARGET mwal_replication)
  add_executable(replicated_node replicated_node.cc)
  target_link_libraries(replicated_node PRIVATE mwal_replication)
  target_include_directories(replicated_node PRIVATE
    ${CMAKE_SOURCE_DIR}/src
    ${CMAKE_SOURCE_DIR}/include
    ${CMAKE_BINARY_DIR}/proto_gen
  )
endif()
