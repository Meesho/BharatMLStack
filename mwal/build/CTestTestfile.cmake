# CMake generated Testfile for 
# Source directory: /Users/anshagrawal/Meesho/BharatMLStack/mwal
# Build directory: /Users/anshagrawal/Meesho/BharatMLStack/mwal/build
# 
# This file includes the relevant testing commands required for 
# testing this directory and lists subdirectories to be tested as well.
add_test(isr_tracker_test "/Users/anshagrawal/Meesho/BharatMLStack/mwal/build/isr_tracker_test")
set_tests_properties(isr_tracker_test PROPERTIES  _BACKTRACE_TRIPLES "/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;199;add_test;/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;202;mwal_add_repl_test;/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;0;")
add_test(replication_manager_test "/Users/anshagrawal/Meesho/BharatMLStack/mwal/build/replication_manager_test")
set_tests_properties(replication_manager_test PROPERTIES  _BACKTRACE_TRIPLES "/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;199;add_test;/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;203;mwal_add_repl_test;/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;0;")
add_test(replication_e2e_test "/Users/anshagrawal/Meesho/BharatMLStack/mwal/build/replication_e2e_test")
set_tests_properties(replication_e2e_test PROPERTIES  _BACKTRACE_TRIPLES "/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;199;add_test;/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;204;mwal_add_repl_test;/Users/anshagrawal/Meesho/BharatMLStack/mwal/CMakeLists.txt;0;")
subdirs("_deps/googletest-build")
subdirs("test")
subdirs("bench")
subdirs("_deps/nuraft-build")
subdirs("examples")
