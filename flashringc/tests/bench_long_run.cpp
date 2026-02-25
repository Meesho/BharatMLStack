#include "flashringc/cache.h"

#include <algorithm>
#include <array>
#include <atomic>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <mutex>
#include <random>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

using Clock = std::chrono::high_resolution_clock;

static constexpr int kMaxWorkers = 256;

struct BenchArgs {
    std::string device_path       = "/tmp/flashring_longrun.dat";
    double      duration_sec      = 300.0;
    double      report_interval   = 15.0;
    uint64_t    num_keys          = 1'000'000;
    int         num_workers       = 1;
    size_t      batch_size        = 100;
    size_t      val_size          = 4096;

    uint64_t    ring_capacity     = 0;
    size_t      memtable_size     = 64ULL << 20;
    uint32_t    index_capacity    = 0;
    uint32_t    num_shards        = 0;
    uint32_t    queue_capacity    = 16384;
    uint32_t    uring_queue_depth = 512;
    size_t      sem_pool_capacity = 4096;
    double      eviction_threshold = 0.95;
    double      clear_threshold    = 0.65;
};

static void print_usage(const char* prog) {
    printf("Usage: %s [options]\n", prog);
    printf("\nBenchmark parameters:\n");
    printf("  --path <str>          Device/file path         (default: /tmp/flashring_longrun.dat)\n");
    printf("  --duration <sec>      Run duration             (default: 300)\n");
    printf("  --interval <sec>      Report interval          (default: 15)\n");
    printf("  --num-keys <n>        Working set size         (default: 1000000)\n");
    printf("  --workers <n>         Worker threads           (default: 1)\n");
    printf("  --batch-size <n>      Keys per batch           (default: 100)\n");
    printf("  --val-size <n>        Value size in bytes      (default: 4096)\n");
    printf("\nCacheConfig parameters:\n");
    printf("  --ring-capacity <bytes>   Ring capacity, 0=auto         (default: auto)\n");
    printf("  --memtable-size <bytes>   Memtable size                 (default: 67108864)\n");
    printf("  --index-capacity <n>      Index capacity, 0=auto        (default: auto)\n");
    printf("  --num-shards <n>          Shard count, 0=auto           (default: 0)\n");
    printf("  --queue-capacity <n>      MPSC queue capacity           (default: 16384)\n");
    printf("  --uring-depth <n>         io_uring queue depth          (default: 512)\n");
    printf("  --sem-pool <n>            Semaphore pool capacity       (default: 4096)\n");
    printf("  --eviction-threshold <f>  Start evicting at this usage  (default: 0.95)\n");
    printf("  --clear-threshold <f>     Stop evicting at this usage   (default: 0.65)\n");
    printf("\nLegacy positional args also supported:\n");
    printf("  %s <path> <duration> <interval> <num_keys> <workers>\n", prog);
}

static BenchArgs parse_args(int argc, char* argv[]) {
    BenchArgs a;

    bool has_flags = false;
    for (int i = 1; i < argc; ++i) {
        if (argv[i][0] == '-') { has_flags = true; break; }
    }

    if (!has_flags) {
        if (argc >= 2) a.device_path = argv[1];
        if (argc >= 3) a.duration_sec = std::max(1.0, atof(argv[2]));
        if (argc >= 4) a.report_interval = std::max(1.0, atof(argv[3]));
        if (argc >= 5) { long long v = atoll(argv[4]); a.num_keys = v > 0 ? static_cast<uint64_t>(v) : 1; }
        if (argc >= 6) { int w = atoi(argv[5]); a.num_workers = w > 0 ? std::min(w, kMaxWorkers) : 1; }
        return a;
    }

    for (int i = 1; i < argc; ++i) {
        auto next = [&]() -> const char* {
            return (i + 1 < argc) ? argv[++i] : nullptr;
        };
        std::string flag = argv[i];
        if (flag == "--help" || flag == "-h") { print_usage(argv[0]); exit(0); }
        else if (flag == "--path")                { auto v = next(); if (v) a.device_path = v; }
        else if (flag == "--duration")            { auto v = next(); if (v) a.duration_sec = std::max(1.0, atof(v)); }
        else if (flag == "--interval")            { auto v = next(); if (v) a.report_interval = std::max(1.0, atof(v)); }
        else if (flag == "--num-keys")            { auto v = next(); if (v) { long long x = atoll(v); a.num_keys = x > 0 ? static_cast<uint64_t>(x) : 1; } }
        else if (flag == "--workers")             { auto v = next(); if (v) { int w = atoi(v); a.num_workers = w > 0 ? std::min(w, kMaxWorkers) : 1; } }
        else if (flag == "--batch-size")           { auto v = next(); if (v) a.batch_size = static_cast<size_t>(std::max(1, atoi(v))); }
        else if (flag == "--val-size")             { auto v = next(); if (v) a.val_size = static_cast<size_t>(std::max(1, atoi(v))); }
        else if (flag == "--ring-capacity")        { auto v = next(); if (v) a.ring_capacity = static_cast<uint64_t>(atoll(v)); }
        else if (flag == "--memtable-size")        { auto v = next(); if (v) a.memtable_size = static_cast<size_t>(atoll(v)); }
        else if (flag == "--index-capacity")       { auto v = next(); if (v) a.index_capacity = static_cast<uint32_t>(atoi(v)); }
        else if (flag == "--num-shards")           { auto v = next(); if (v) a.num_shards = static_cast<uint32_t>(atoi(v)); }
        else if (flag == "--queue-capacity")       { auto v = next(); if (v) a.queue_capacity = static_cast<uint32_t>(atoi(v)); }
        else if (flag == "--uring-depth")          { auto v = next(); if (v) a.uring_queue_depth = static_cast<uint32_t>(atoi(v)); }
        else if (flag == "--sem-pool")             { auto v = next(); if (v) a.sem_pool_capacity = static_cast<size_t>(atoi(v)); }
        else if (flag == "--eviction-threshold")   { auto v = next(); if (v) a.eviction_threshold = atof(v); }
        else if (flag == "--clear-threshold")       { auto v = next(); if (v) a.clear_threshold = atof(v); }
        else { fprintf(stderr, "Unknown flag: %s\n", argv[i]); print_usage(argv[0]); exit(1); }
    }
    return a;
}

static void cleanup(const std::string& path) {
    for (uint32_t i = 0; i < 256; ++i)
        ::unlink((path + "." + std::to_string(i)).c_str());
}

static void compute_percentiles(std::vector<double>& lats,
                                double* avg, double* p50, double* p90,
                                double* p99, double* p999) {
    if (lats.empty()) {
        *avg = *p50 = *p90 = *p99 = *p999 = 0;
        return;
    }
    std::sort(lats.begin(), lats.end());
    size_t n = lats.size();
    double sum = 0;
    for (double x : lats) sum += x;
    *avg  = sum / static_cast<double>(n);
    *p50  = lats[static_cast<size_t>(static_cast<double>(n) * 0.50)];
    *p90  = lats[static_cast<size_t>(static_cast<double>(n) * 0.90)];
    *p99  = lats[std::min(static_cast<size_t>(static_cast<double>(n) * 0.99), n - 1)];
    *p999 = lats[std::min(static_cast<size_t>(static_cast<double>(n) * 0.999), n - 1)];
}

static std::string run_command(const char* cmd) {
    std::string result;
    std::array<char, 256> buf;
    FILE* pipe = popen(cmd, "r");
    if (!pipe) return "(popen failed)";
    while (fgets(buf.data(), static_cast<int>(buf.size()), pipe))
        result += buf.data();
    pclose(pipe);
    while (!result.empty() && result.back() == '\n') result.pop_back();
    return result;
}

static std::string extract_device_name(const std::string& path) {
    size_t last_slash = path.rfind('/');
    if (last_slash == std::string::npos) return path;
    return path.substr(last_slash + 1);
}

int main(int argc, char* argv[]) {
    BenchArgs args = parse_args(argc, argv);

    if (args.ring_capacity == 0) {
        const uint64_t working_set_bytes = args.num_keys * 4096ULL;
        args.ring_capacity = (working_set_bytes * 4ULL) / 3ULL;
        if (args.ring_capacity < 512ULL * 1024 * 1024) args.ring_capacity = 512ULL * 1024 * 1024;
        if (args.ring_capacity > 8ULL * 1024 * 1024 * 1024) args.ring_capacity = 8ULL * 1024 * 1024 * 1024;
    }
    if (args.index_capacity == 0) {
        args.index_capacity = static_cast<uint32_t>(std::min(args.num_keys * 2ULL, 4'000'000ULL));
    }

    cleanup(args.device_path);

    CacheConfig cfg{
        .device_path        = args.device_path,
        .ring_capacity      = args.ring_capacity,
        .memtable_size      = args.memtable_size,
        .index_capacity     = args.index_capacity,
        .num_shards         = args.num_shards,
        .queue_capacity     = args.queue_capacity,
        .uring_queue_depth  = args.uring_queue_depth,
        .sem_pool_capacity  = args.sem_pool_capacity,
        .eviction_threshold = args.eviction_threshold,
        .clear_threshold    = args.clear_threshold,
    };

    // --- System info ---
    printf("=== System Info ===\n");
    printf("  nproc:           %s\n", run_command("nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null").c_str());
    printf("  CPU:             %s\n", run_command("lscpu 2>/dev/null | grep 'Model name' | sed 's/.*: *//' || sysctl -n machdep.cpu.brand_string 2>/dev/null").c_str());
    printf("  kernel:          %s\n", run_command("uname -r").c_str());

    bool is_block_device = (args.device_path.find("/dev/") == 0);
    if (is_block_device) {
        std::string dev_name = extract_device_name(args.device_path);
        std::string sched_cmd = "cat /sys/block/" + dev_name + "/queue/scheduler 2>/dev/null";
        std::string nr_req_cmd = "cat /sys/block/" + dev_name + "/queue/nr_requests 2>/dev/null";
        printf("  block_device:    %s\n", dev_name.c_str());
        printf("  io_scheduler:    %s\n", run_command(sched_cmd.c_str()).c_str());
        printf("  nr_requests:     %s\n", run_command(nr_req_cmd.c_str()).c_str());
    }

    // --- Config dump ---
    printf("\n=== Benchmark Config ===\n");
    printf("  path:              %s\n", args.device_path.c_str());
    printf("  duration:          %.0fs\n", args.duration_sec);
    printf("  report_interval:   %.0fs\n", args.report_interval);
    printf("  num_keys:          %lu\n", (unsigned long)args.num_keys);
    printf("  workers:           %d\n", args.num_workers);
    printf("  batch_size:        %zu\n", args.batch_size);
    printf("  val_size:          %zu\n", args.val_size);

    printf("\n=== CacheConfig ===\n");
    printf("  ring_capacity:     %lu (%.2f GB)\n", (unsigned long)cfg.ring_capacity, static_cast<double>(cfg.ring_capacity) / (1024.0*1024*1024));
    printf("  memtable_size:     %zu (%.2f MB)\n", cfg.memtable_size, static_cast<double>(cfg.memtable_size) / (1024.0*1024));
    printf("  index_capacity:    %u\n", cfg.index_capacity);
    printf("  num_shards:        %u (0=auto)\n", cfg.num_shards);
    printf("  queue_capacity:    %u\n", cfg.queue_capacity);
    printf("  uring_queue_depth: %u\n", cfg.uring_queue_depth);
    printf("  sem_pool_capacity: %zu\n", cfg.sem_pool_capacity);
    printf("  eviction_thresh:   %.2f\n", cfg.eviction_threshold);
    printf("  clear_thresh:      %.2f\n", cfg.clear_threshold);

    auto cache = Cache::open(cfg);
    uint32_t actual_shards = cache.num_shards();
    printf("  actual_shards:     %u\n", actual_shards);
    printf("\n");

    std::string val(args.val_size, 'V');

    std::atomic<uint64_t> total_gets{0};
    std::atomic<uint64_t> total_puts{0};
    std::atomic<uint64_t> total_hits{0};
    std::atomic<uint64_t> total_batch_ops{0};

    std::mutex lat_mu;
    std::vector<double> window_latencies;
    window_latencies.reserve(100000);

    std::atomic<bool> stop{false};
    auto t_start = Clock::now();
    auto deadline = t_start + std::chrono::duration<double>(args.duration_sec);

    // --- iostat thread ---
    std::string iostat_dev = is_block_device ? extract_device_name(args.device_path) : "";
    std::thread iostat_thread;
    if (!iostat_dev.empty()) {
        iostat_thread = std::thread([&]() {
            while (!stop.load(std::memory_order_relaxed)) {
                std::this_thread::sleep_for(
                    std::chrono::milliseconds(static_cast<int>(args.report_interval * 1000)));
                if (stop.load(std::memory_order_relaxed)) break;

                std::string cmd = "iostat -x -d " + iostat_dev + " 1 2 2>/dev/null | tail -1";
                std::string out = run_command(cmd.c_str());
                double elapsed = std::chrono::duration<double>(Clock::now() - t_start).count();
                printf("[iostat %3.0fs] %s\n", elapsed, out.c_str());
            }
        });
    }

    // --- Reporter thread ---
    std::thread reporter([&]() {
        uint64_t prev_gets = 0, prev_puts = 0, prev_batch_ops = 0, prev_hits = 0;
        uint64_t prev_wrap = 0;

        while (!stop.load(std::memory_order_relaxed)) {
            std::this_thread::sleep_for(
                std::chrono::milliseconds(static_cast<int>(args.report_interval * 1000)));
            if (stop.load(std::memory_order_relaxed)) break;

            uint64_t gets = total_gets.load(std::memory_order_relaxed);
            uint64_t puts = total_puts.load(std::memory_order_relaxed);
            uint64_t batch_ops = total_batch_ops.load(std::memory_order_relaxed);
            uint64_t hits = total_hits.load(std::memory_order_relaxed);
            uint64_t wrap = cache.ring_wrap_count();
            uint32_t kc = cache.key_count();
            uint64_t ring_used = cache.ring_usage();

            uint64_t d_gets = gets - prev_gets;
            uint64_t d_puts = puts - prev_puts;
            uint64_t d_batch = batch_ops - prev_batch_ops;
            uint64_t d_hits = hits - prev_hits;
            uint64_t d_wrap = wrap - prev_wrap;

            double window_hit_rate = (d_gets > 0)
                ? 100.0 * static_cast<double>(d_hits) / static_cast<double>(d_gets) : 0.0;

            double avg_us = 0, p50_us = 0, p90_us = 0, p99_us = 0, p999_us = 0;
            size_t lat_count = 0;
            {
                std::lock_guard<std::mutex> lock(lat_mu);
                lat_count = window_latencies.size();
                compute_percentiles(window_latencies, &avg_us, &p50_us, &p90_us, &p99_us, &p999_us);
                window_latencies.clear();
            }

            double elapsed = std::chrono::duration<double>(Clock::now() - t_start).count();
            double ring_util = (cfg.ring_capacity > 0)
                ? 100.0 * static_cast<double>(ring_used) / static_cast<double>(cfg.ring_capacity) : 0.0;

            printf("[%3.0fs] gets=%lu puts=%lu batch_ops=%lu | "
                   "lat_avg=%.1fus p50=%.1fus p90=%.1fus p99=%.1fus p999=%.1fus (n=%zu) | "
                   "hit_rate=%.1f%% keys=%u ring_util=%.1f%% wrap=%lu\n",
                   elapsed,
                   (unsigned long)d_gets, (unsigned long)d_puts, (unsigned long)d_batch,
                   avg_us, p50_us, p90_us, p99_us, p999_us, lat_count,
                   window_hit_rate, kc, ring_util, (unsigned long)d_wrap);

            prev_gets = gets;
            prev_puts = puts;
            prev_batch_ops = batch_ops;
            prev_hits = hits;
            prev_wrap = wrap;
        }
    });

    // --- Worker threads ---
    auto worker_loop = [&](int thread_index) {
        std::mt19937 rng(12345 + static_cast<unsigned>(thread_index));
        std::uniform_int_distribution<uint64_t> key_dist(0, args.num_keys - 1);
        std::vector<std::string> key_storage(args.batch_size);
        std::vector<std::string_view> keys(args.batch_size);
        std::vector<KVPair> put_pairs;
        put_pairs.reserve(args.batch_size);

        while (Clock::now() < deadline) {
            for (size_t i = 0; i < args.batch_size; ++i) {
                uint64_t k = key_dist(rng);
                key_storage[i] = "lk_" + std::to_string(k);
                keys[i] = key_storage[i];
            }

            auto t0 = Clock::now();
            std::vector<Result> results = cache.batch_get(keys);
            uint64_t hits_this_batch = 0;
            put_pairs.clear();
            for (size_t i = 0; i < results.size(); ++i) {
                if (results[i].status == Status::Ok)
                    ++hits_this_batch;
                else if (results[i].status == Status::NotFound)
                    put_pairs.push_back(KVPair{keys[i], std::string_view(val)});
            }
            if (!put_pairs.empty())
                cache.batch_put(put_pairs);
            auto t1 = Clock::now();

            double us = std::chrono::duration<double, std::micro>(t1 - t0).count();
            {
                std::lock_guard<std::mutex> lock(lat_mu);
                window_latencies.push_back(us);
            }

            total_gets.fetch_add(static_cast<uint64_t>(args.batch_size), std::memory_order_relaxed);
            total_puts.fetch_add(static_cast<uint64_t>(put_pairs.size()), std::memory_order_relaxed);
            total_hits.fetch_add(hits_this_batch, std::memory_order_relaxed);
            total_batch_ops.fetch_add(1, std::memory_order_relaxed);
        }
    };

    std::vector<std::thread> workers;
    workers.reserve(static_cast<size_t>(args.num_workers));
    for (int i = 0; i < args.num_workers; ++i)
        workers.emplace_back(worker_loop, i);
    for (std::thread& t : workers)
        t.join();

    stop.store(true, std::memory_order_release);
    reporter.join();
    if (iostat_thread.joinable())
        iostat_thread.join();

    // --- Final summary ---
    uint64_t gets = total_gets.load();
    uint64_t puts = total_puts.load();
    uint64_t batch_ops = total_batch_ops.load();
    uint64_t hits = total_hits.load();
    double hit_rate = gets > 0 ? 100.0 * static_cast<double>(hits) / static_cast<double>(gets) : 0.0;
    double total_elapsed = std::chrono::duration<double>(Clock::now() - t_start).count();

    printf("\n=== Final Summary (%.1fs) ===\n", total_elapsed);
    printf("  gets=%lu puts=%lu batch_ops=%lu\n",
           (unsigned long)gets, (unsigned long)puts, (unsigned long)batch_ops);
    printf("  hit_rate=%.2f%% keys=%u ring_wrap=%lu\n",
           hit_rate, cache.key_count(), (unsigned long)cache.ring_wrap_count());
    printf("  throughput: %.0f gets/s  %.0f puts/s  %.0f batch_ops/s\n",
           static_cast<double>(gets) / total_elapsed,
           static_cast<double>(puts) / total_elapsed,
           static_cast<double>(batch_ops) / total_elapsed);

    cleanup(args.device_path);
    return 0;
}
