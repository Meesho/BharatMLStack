#pragma once

#include <cstdlib>
#include <filesystem>
#include <string>

#include "mwal/env.h"

namespace mwal {
namespace test {

class TempDir {
 public:
  TempDir() {
    char tmpl[] = "/tmp/mwal_test_XXXXXX";
    char* dir = mkdtemp(tmpl);
    path_ = dir;
  }
  ~TempDir() { std::filesystem::remove_all(path_); }
  const std::string& path() const { return path_; }

 private:
  std::string path_;
};

inline std::string TempFileName(const std::string& dir, uint64_t log_num) {
  char buf[64];
  snprintf(buf, sizeof(buf), "%06llu.log", (unsigned long long)log_num);
  return dir + "/" + buf;
}

}  // namespace test
}  // namespace mwal
