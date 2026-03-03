// Derived from RocksDB — adapted for mwal standalone WAL library.

#pragma once

#include <cstdint>
#include <mutex>
#include <condition_variable>
#include <thread>

namespace mwal {
namespace port {

#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__ || defined(_LITTLE_ENDIAN) || \
    defined(__LITTLE_ENDIAN__)
static constexpr bool kLittleEndian = true;
#else
static constexpr bool kLittleEndian = false;
#endif

using Mutex = std::mutex;
using CondVar = std::condition_variable;

}  // namespace port
}  // namespace mwal
