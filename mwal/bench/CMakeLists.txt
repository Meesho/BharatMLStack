function(mwal_add_bench name)
  add_executable(${name} ${name}.cc)
  target_link_libraries(${name} PRIVATE mwal benchmark::benchmark benchmark::benchmark_main)
  target_include_directories(${name} PRIVATE
    ${CMAKE_SOURCE_DIR}/src
    ${CMAKE_SOURCE_DIR}/include
    ${CMAKE_SOURCE_DIR}/bench
  )
endfunction()

mwal_add_bench(wal_write_bench)
mwal_add_bench(wal_read_bench)
mwal_add_bench(wal_sync_bench)
mwal_add_bench(db_wal_bench)
