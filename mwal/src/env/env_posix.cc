// Derived from RocksDB — POSIX Env implementation for mwal standalone WAL library.

#include "mwal/env.h"

#include <cerrno>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <dirent.h>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

namespace mwal {

namespace {

class PosixFileLock : public FileLock {
 public:
  explicit PosixFileLock(int fd, std::string path)
      : fd_(fd), path_(std::move(path)) {}
  ~PosixFileLock() override {
    if (fd_ >= 0) {
      ::flock(fd_, LOCK_UN);
      ::close(fd_);
    }
  }
  int fd() const { return fd_; }
  const std::string& path() const { return path_; }

 private:
  int fd_;
  std::string path_;
};

class PosixWritableFile : public WritableFile {
 public:
  PosixWritableFile(const std::string& fname, int fd)
      : filename_(fname), fd_(fd), filesize_(0) {}

  ~PosixWritableFile() override {
    if (fd_ >= 0) {
      close(fd_);
    }
  }

  IOStatus Append(const Slice& data) override {
    const char* src = data.data();
    size_t left = data.size();
    while (left > 0) {
      ssize_t done = write(fd_, src, left);
      if (done < 0) {
        if (errno == EINTR) continue;
        return IOStatus::IOError(filename_, strerror(errno));
      }
      left -= done;
      src += done;
    }
    filesize_ += data.size();
    return IOStatus::OK();
  }

  IOStatus Close() override {
    if (fd_ >= 0) {
      if (close(fd_) != 0) {
        return IOStatus::IOError(filename_, strerror(errno));
      }
      fd_ = -1;
    }
    return IOStatus::OK();
  }

  IOStatus Flush() override { return IOStatus::OK(); }

  IOStatus Sync() override {
#ifdef __APPLE__
    if (fcntl(fd_, F_FULLFSYNC) < 0) {
      return IOStatus::IOError(filename_, strerror(errno));
    }
#else
    if (fdatasync(fd_) != 0) {
      return IOStatus::IOError(filename_, strerror(errno));
    }
#endif
    return IOStatus::OK();
  }

  uint64_t GetFileSize() const override { return filesize_; }

  IOStatus Truncate(uint64_t size) override {
    if (ftruncate(fd_, static_cast<off_t>(size)) != 0) {
      return IOStatus::IOError(filename_, strerror(errno));
    }
    filesize_ = size;
    return IOStatus::OK();
  }

 private:
  std::string filename_;
  int fd_;
  uint64_t filesize_;
};

class PosixSequentialFile : public SequentialFile {
 public:
  PosixSequentialFile(const std::string& fname, FILE* file)
      : filename_(fname), file_(file) {}

  ~PosixSequentialFile() override {
    if (file_) fclose(file_);
  }

  IOStatus Read(size_t n, Slice* result, char* scratch) override {
    size_t r = fread(scratch, 1, n, file_);
    *result = Slice(scratch, r);
    if (r < n) {
      if (!feof(file_)) {
        return IOStatus::IOError(filename_, strerror(errno));
      }
    }
    return IOStatus::OK();
  }

  IOStatus Skip(uint64_t n) override {
    if (fseek(file_, static_cast<long>(n), SEEK_CUR)) {
      return IOStatus::IOError(filename_, strerror(errno));
    }
    return IOStatus::OK();
  }

 private:
  std::string filename_;
  FILE* file_;
};

class PosixEnv : public Env {
 public:
  PosixEnv() = default;

  Status NewWritableFile(const std::string& fname,
                         std::unique_ptr<WritableFile>* result,
                         const EnvOptions& /*options*/) override {
    int fd = open(fname.c_str(), O_CREAT | O_RDWR | O_TRUNC, 0644);
    if (fd < 0) {
      *result = nullptr;
      return Status::IOError(fname, strerror(errno));
    }
    result->reset(new PosixWritableFile(fname, fd));
    return Status::OK();
  }

  Status NewSequentialFile(const std::string& fname,
                           std::unique_ptr<SequentialFile>* result,
                           const EnvOptions& /*options*/) override {
    FILE* file = fopen(fname.c_str(), "r");
    if (file == nullptr) {
      *result = nullptr;
      return Status::IOError(fname, strerror(errno));
    }
    result->reset(new PosixSequentialFile(fname, file));
    return Status::OK();
  }

  Status DeleteFile(const std::string& fname) override {
    if (unlink(fname.c_str()) != 0) {
      return Status::IOError(fname, strerror(errno));
    }
    return Status::OK();
  }

  Status RenameFile(const std::string& src,
                    const std::string& target) override {
    if (rename(src.c_str(), target.c_str()) != 0) {
      return Status::IOError(src, strerror(errno));
    }
    return Status::OK();
  }

  Status GetChildren(const std::string& dir,
                     std::vector<std::string>* result) override {
    result->clear();
    DIR* d = opendir(dir.c_str());
    if (d == nullptr) {
      return Status::IOError(dir, strerror(errno));
    }
    struct dirent* entry;
    while ((entry = readdir(d)) != nullptr) {
      result->push_back(entry->d_name);
    }
    closedir(d);
    return Status::OK();
  }

  Status FileExists(const std::string& fname) override {
    if (access(fname.c_str(), F_OK) == 0) {
      return Status::OK();
    }
    return Status::NotFound(fname, strerror(errno));
  }

  Status GetFileSize(const std::string& fname, uint64_t* size) override {
    struct stat sbuf;
    if (stat(fname.c_str(), &sbuf) != 0) {
      *size = 0;
      return Status::IOError(fname, strerror(errno));
    }
    *size = static_cast<uint64_t>(sbuf.st_size);
    return Status::OK();
  }

  Status GetFileModificationTime(const std::string& fname,
                                   uint64_t* file_mtime) override {
    struct stat sbuf;
    if (stat(fname.c_str(), &sbuf) != 0) {
      *file_mtime = 0;
      return Status::IOError(fname, strerror(errno));
    }
    *file_mtime = static_cast<uint64_t>(sbuf.st_mtime);
    return Status::OK();
  }

  Status CreateDir(const std::string& dirname) override {
    if (mkdir(dirname.c_str(), 0755) != 0) {
      return Status::IOError(dirname, strerror(errno));
    }
    return Status::OK();
  }

  Status CreateDirIfMissing(const std::string& dirname) override {
    if (mkdir(dirname.c_str(), 0755) != 0) {
      if (errno != EEXIST) {
        return Status::IOError(dirname, strerror(errno));
      }
    }
    return Status::OK();
  }

  uint64_t NowMicros() override {
    auto now = std::chrono::system_clock::now();
    return static_cast<uint64_t>(
        std::chrono::duration_cast<std::chrono::microseconds>(
            now.time_since_epoch())
            .count());
  }

  Status LockFile(const std::string& fname, FileLock** lock) override {
    *lock = nullptr;
    int fd = ::open(fname.c_str(), O_RDWR | O_CREAT, 0644);
    if (fd < 0) {
      return Status::IOError(fname, strerror(errno));
    }
    if (::flock(fd, LOCK_EX | LOCK_NB) != 0) {
      ::close(fd);
      return Status::IOError(fname, strerror(errno));
    }
    *lock = new PosixFileLock(fd, fname);
    return Status::OK();
  }

  Status UnlockFile(FileLock* lock) override {
    if (!lock) return Status::OK();
    auto* posix_lock = static_cast<PosixFileLock*>(lock);
    int fd = posix_lock->fd();
    if (fd >= 0) {
      ::flock(fd, LOCK_UN);
      ::close(fd);
    }
    std::string path = posix_lock->path();
    delete posix_lock;
    ::unlink(path.c_str());
    return Status::OK();
  }
};

}  // namespace

Env* Env::Default() {
  static PosixEnv default_env;
  return &default_env;
}

}  // namespace mwal
