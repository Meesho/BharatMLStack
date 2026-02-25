#ifndef EIGENIX_BENCH_UTILS_HPP
#define EIGENIX_BENCH_UTILS_HPP

#include <chrono>
#include <cstdio>
#include <fstream>
#include <string>
#include <vector>

#ifdef __linux__
#include <fstream>
#include <sstream>
#endif

#ifdef __APPLE__
#include <mach/mach.h>
#endif

#if !defined(__APPLE__)
#include <sys/resource.h>
#endif

namespace eigenix {

// High-resolution wall-clock timer (milliseconds).
class ScopedTimer {
public:
    ScopedTimer() : start_(std::chrono::steady_clock::now()) {}
    void reset() { start_ = std::chrono::steady_clock::now(); }
    double elapsed_ms() const {
        auto now = std::chrono::steady_clock::now();
        return std::chrono::duration<double, std::milli>(now - start_).count();
    }
private:
    std::chrono::steady_clock::time_point start_;
};

// Peak resident set size in MB.
inline double get_peak_rss_mb() {
#ifdef __linux__
    std::ifstream f("/proc/self/status");
    std::string line;
    while (std::getline(f, line)) {
        if (line.rfind("VmHWM:", 0) == 0) {
            // VmHWM: <value> kB
            long kb = 0;
            std::sscanf(line.c_str(), "VmHWM: %ld kB", &kb);
            return static_cast<double>(kb) / 1024.0;
        }
    }
    return 0.0;
#elif defined(__APPLE__)
    struct mach_task_basic_info info;
    mach_msg_type_number_t count = MACH_TASK_BASIC_INFO_COUNT;
    if (task_info(mach_task_self(), MACH_TASK_BASIC_INFO,
                  reinterpret_cast<task_info_t>(&info), &count) == KERN_SUCCESS) {
        return static_cast<double>(info.resident_size_max) / (1024.0 * 1024.0);
    }
    return 0.0;
#else
    struct rusage ru;
    getrusage(RUSAGE_SELF, &ru);
    return static_cast<double>(ru.ru_maxrss) / 1024.0;
#endif
}

// Check Linux CPU governor; warn if not "performance".
inline void check_cpu_governor() {
#ifdef __linux__
    std::ifstream f("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor");
    if (f.is_open()) {
        std::string gov;
        std::getline(f, gov);
        if (gov != "performance") {
            std::fprintf(stderr,
                "WARNING: CPU governor is '%s', not 'performance'. "
                "Results may be noisy.\n", gov.c_str());
        }
    }
#endif
}

// Simple CSV writer.
class CsvWriter {
public:
    explicit CsvWriter(const std::string& path) : out_(path) {}
    bool is_open() const { return out_.is_open(); }

    void write_header(const std::vector<std::string>& cols) {
        for (size_t i = 0; i < cols.size(); ++i) {
            if (i > 0) out_ << ',';
            out_ << cols[i];
        }
        out_ << '\n';
    }

    template <typename... Args>
    void write_row(Args&&... args) {
        write_vals(std::forward<Args>(args)...);
        out_ << '\n';
    }

    void flush() { out_.flush(); }

private:
    std::ofstream out_;

    template <typename T>
    void write_vals(T&& v) { out_ << v; }

    template <typename T, typename... Rest>
    void write_vals(T&& v, Rest&&... rest) {
        out_ << v << ',';
        write_vals(std::forward<Rest>(rest)...);
    }
};

struct BenchResult {
    std::string backend;
    size_t n;
    double train_time_ms;
    double assign_time_ms;
    float inertia;
    int iterations;
    double throughput_mvecs_per_sec;
    double memory_mb;
    int cluster_size_min;
    int cluster_size_max;
    float cluster_size_stddev;
    int empty_clusters;
};

}  // namespace eigenix

#endif  // EIGENIX_BENCH_UTILS_HPP
