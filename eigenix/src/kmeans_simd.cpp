#include "kmeans_simd.hpp"
#include "metrics.hpp"
#include <cmath>
#include <cstring>
#include <limits>
#include <omp.h>
#include <random>
#include <stdexcept>

#if defined(__x86_64__) || defined(_M_X64)
#include <immintrin.h>
#ifdef _MSC_VER
#include <intrin.h>
#else
#include <cpuid.h>
#endif
#endif

namespace eigenix {

// ============== CPUID ISA detection ==============

namespace {

#if defined(__x86_64__) || defined(_M_X64)

struct CpuIdResult { unsigned eax, ebx, ecx, edx; };

inline CpuIdResult cpuid(unsigned leaf, unsigned sub = 0) {
    CpuIdResult r{};
#ifdef _MSC_VER
    int regs[4];
    __cpuidex(regs, static_cast<int>(leaf), static_cast<int>(sub));
    r.eax = regs[0]; r.ebx = regs[1]; r.ecx = regs[2]; r.edx = regs[3];
#else
    __cpuid_count(leaf, sub, r.eax, r.ebx, r.ecx, r.edx);
#endif
    return r;
}

SimdISA detect_isa() {
    auto r1 = cpuid(1);
    auto r7 = cpuid(7, 0);
    bool has_avx2 = (r7.ebx >> 5) & 1;
    bool has_fma  = (r1.ecx >> 12) & 1;
    bool has_avx512f = (r7.ebx >> 16) & 1;

    if (has_avx512f) return SimdISA::AVX512;
    if (has_avx2 && has_fma) return SimdISA::AVX2;
    return SimdISA::SSE42;
}

#else

SimdISA detect_isa() { return SimdISA::SSE42; }

#endif

}  // namespace

// ============== L2 squared distance kernels ==============

#if defined(__x86_64__) || defined(_M_X64)

static float l2sq_sse42(const float* a, const float* b, int dim) {
    __m128 sum = _mm_setzero_ps();
    int i = 0;
    for (; i + 4 <= dim; i += 4) {
        __m128 va = _mm_loadu_ps(a + i);
        __m128 vb = _mm_loadu_ps(b + i);
        __m128 d  = _mm_sub_ps(va, vb);
        sum = _mm_add_ps(sum, _mm_mul_ps(d, d));
    }
    // Horizontal sum
    __m128 shuf = _mm_movehdup_ps(sum);
    sum = _mm_add_ps(sum, shuf);
    shuf = _mm_movehl_ps(shuf, sum);
    sum = _mm_add_ss(sum, shuf);
    float result = _mm_cvtss_f32(sum);
    for (; i < dim; ++i) {
        float d = a[i] - b[i];
        result += d * d;
    }
    return result;
}

#ifdef __AVX2__
__attribute__((target("avx2,fma")))
static float l2sq_avx2(const float* a, const float* b, int dim) {
    __m256 sum = _mm256_setzero_ps();
    int i = 0;
    for (; i + 8 <= dim; i += 8) {
        _mm_prefetch(reinterpret_cast<const char*>(a + i + 64), _MM_HINT_T0);
        _mm_prefetch(reinterpret_cast<const char*>(b + i + 64), _MM_HINT_T0);
        __m256 va = _mm256_loadu_ps(a + i);
        __m256 vb = _mm256_loadu_ps(b + i);
        __m256 d  = _mm256_sub_ps(va, vb);
        sum = _mm256_fmadd_ps(d, d, sum);
    }
    // Horizontal sum 256→scalar
    __m128 lo = _mm256_castps256_ps128(sum);
    __m128 hi = _mm256_extractf128_ps(sum, 1);
    __m128 s  = _mm_add_ps(lo, hi);
    __m128 shuf = _mm_movehdup_ps(s);
    s = _mm_add_ps(s, shuf);
    shuf = _mm_movehl_ps(shuf, s);
    s = _mm_add_ss(s, shuf);
    float result = _mm_cvtss_f32(s);
    for (; i < dim; ++i) {
        float d = a[i] - b[i];
        result += d * d;
    }
    return result;
}
#endif

#ifdef __AVX512F__
__attribute__((target("avx512f")))
static float l2sq_avx512(const float* a, const float* b, int dim) {
    __m512 sum = _mm512_setzero_ps();
    int i = 0;
    for (; i + 16 <= dim; i += 16) {
        _mm_prefetch(reinterpret_cast<const char*>(a + i + 64), _MM_HINT_T0);
        _mm_prefetch(reinterpret_cast<const char*>(b + i + 64), _MM_HINT_T0);
        __m512 va = _mm512_loadu_ps(a + i);
        __m512 vb = _mm512_loadu_ps(b + i);
        __m512 d  = _mm512_sub_ps(va, vb);
        sum = _mm512_fmadd_ps(d, d, sum);
    }
    float result = _mm512_reduce_add_ps(sum);
    for (; i < dim; ++i) {
        float d = a[i] - b[i];
        result += d * d;
    }
    return result;
}
#endif

#else  // Non-x86

static float l2sq_scalar(const float* a, const float* b, int dim) {
    float s = 0.0f;
    for (int i = 0; i < dim; ++i) {
        float d = a[i] - b[i];
        s += d * d;
    }
    return s;
}

#endif  // x86_64

// ============== SimdKMeans implementation ==============

SimdKMeans::SimdKMeans() : isa_(detect_isa()) {}

float SimdKMeans::l2sq(const float* a, const float* b, int dim) const {
#if defined(__x86_64__) || defined(_M_X64)
    switch (isa_) {
#ifdef __AVX512F__
        case SimdISA::AVX512: return l2sq_avx512(a, b, dim);
#endif
#ifdef __AVX2__
        case SimdISA::AVX2:   return l2sq_avx2(a, b, dim);
#endif
        default:              return l2sq_sse42(a, b, dim);
    }
#else
    return l2sq_scalar(a, b, dim);
#endif
}

void SimdKMeans::kmeanspp_init(const float* data, size_t n, int dim, int k,
                               std::mt19937& rng) {
    centroids_.resize(static_cast<size_t>(k) * dim);

    std::uniform_int_distribution<size_t> uidx(0, n - 1);
    size_t first = uidx(rng);
    std::memcpy(centroids_.data(), data + first * dim,
                static_cast<size_t>(dim) * sizeof(float));

    std::vector<float> min_dist(n);
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        min_dist[i] = l2sq(data + i * dim, centroids_.data(), dim);

    for (int cc = 1; cc < k; ++cc) {
        float total = 0.0f;
        for (size_t i = 0; i < n; ++i) total += min_dist[i];
        if (total <= 0.0f) total = 1.0f;

        std::uniform_real_distribution<float> u(0.0f, total);
        float r = u(rng);
        size_t chosen = 0;
        for (; chosen < n && r >= 0.0f; ++chosen) r -= min_dist[chosen];
        if (chosen > 0) chosen--;
        chosen = std::min(chosen, n - 1);

        std::memcpy(centroids_.data() + static_cast<size_t>(cc) * dim,
                     data + chosen * dim,
                     static_cast<size_t>(dim) * sizeof(float));

        const float* c_new = centroids_.data() + static_cast<size_t>(cc) * dim;
        #pragma omp parallel for schedule(static)
        for (size_t i = 0; i < n; ++i) {
            float d = l2sq(data + i * dim, c_new, dim);
            if (d < min_dist[i]) min_dist[i] = d;
        }
    }
}

void SimdKMeans::train(const float* data, size_t n, int dim, int k,
                       const TrainConfig& cfg) {
    if (!data || n == 0 || dim <= 0 || k <= 0)
        throw std::invalid_argument("SimdKMeans::train: invalid arguments");
    if (n < static_cast<size_t>(k))
        throw std::invalid_argument("SimdKMeans::train: n must be >= k");

    k_ = k;
    dim_ = dim;
    iterations_ = 0;
    inertia_ = 0.0f;

    std::mt19937 rng(cfg.seed);
    kmeanspp_init(data, n, dim, k, rng);

    std::vector<int> labels(n);
    std::vector<float> centroid_sums(static_cast<size_t>(k) * dim);
    std::vector<size_t> centroid_counts(static_cast<size_t>(k));

    for (size_t iter = 0; iter < cfg.max_iter; ++iter) {
        // Assignment: find nearest centroid for each point.
        #pragma omp parallel for schedule(static)
        for (size_t i = 0; i < n; ++i) {
            const float* x = data + i * dim;
            int best = 0;
            float best_d = l2sq(x, centroids_.data(), dim);
            for (int c = 1; c < k; ++c) {
                float d = l2sq(x, centroids_.data() + static_cast<size_t>(c) * dim, dim);
                if (d < best_d) { best_d = d; best = c; }
            }
            labels[i] = best;
        }

        // Centroid update with thread-local accumulators.
        std::memset(centroid_sums.data(), 0,
                    static_cast<size_t>(k) * dim * sizeof(float));
        std::memset(centroid_counts.data(), 0,
                    static_cast<size_t>(k) * sizeof(size_t));

        #pragma omp parallel
        {
            std::vector<float> lsums(static_cast<size_t>(k) * dim, 0.0f);
            std::vector<size_t> lcounts(static_cast<size_t>(k), 0);
            #pragma omp for schedule(static)
            for (size_t i = 0; i < n; ++i) {
                int l = labels[i];
                const float* x = data + i * dim;
                float* s = lsums.data() + static_cast<size_t>(l) * dim;
                for (int j = 0; j < dim; ++j) s[j] += x[j];
                lcounts[l]++;
            }
            #pragma omp critical
            {
                for (int c = 0; c < k; ++c) {
                    centroid_counts[c] += lcounts[c];
                    const float* src = lsums.data() + static_cast<size_t>(c) * dim;
                    float* dst = centroid_sums.data() + static_cast<size_t>(c) * dim;
                    for (int j = 0; j < dim; ++j) dst[j] += src[j];
                }
            }
        }

        float max_shift = 0.0f;
        for (int c = 0; c < k; ++c) {
            size_t cnt = centroid_counts[c];
            float* cv = centroids_.data() + static_cast<size_t>(c) * dim;
            if (cnt == 0) continue;
            float inv = 1.0f / static_cast<float>(cnt);
            float shift = 0.0f;
            for (int j = 0; j < dim; ++j) {
                float old_c = cv[j];
                float new_c = centroid_sums[static_cast<size_t>(c) * dim + j] * inv;
                cv[j] = new_c;
                float d = new_c - old_c;
                shift += d * d;
            }
            shift = std::sqrt(shift);
            if (shift > max_shift) max_shift = shift;
        }

        iterations_ = static_cast<int>(iter + 1);
        if (max_shift <= cfg.tol) break;
    }

    inertia_ = compute_inertia(data, n, dim, labels.data(),
                               centroids_.data(), k);
}

void SimdKMeans::assign(const float* data, size_t n, int dim,
                        std::vector<int>& labels) const {
    if (!data || dim != dim_ || k_ == 0)
        throw std::runtime_error("SimdKMeans::assign: not trained or dim mismatch");

    labels.resize(n);
    if (n == 0) return;

    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i) {
        const float* x = data + i * dim;
        int best = 0;
        float best_d = l2sq(x, centroids_.data(), dim);
        for (int c = 1; c < k_; ++c) {
            float d = l2sq(x, centroids_.data() + static_cast<size_t>(c) * dim, dim);
            if (d < best_d) { best_d = d; best = c; }
        }
        labels[i] = best;
    }
}

const float* SimdKMeans::centroids() const { return centroids_.data(); }
float SimdKMeans::inertia() const { return inertia_; }
int SimdKMeans::iterations() const { return iterations_; }

std::string SimdKMeans::name() const {
    switch (isa_) {
        case SimdISA::AVX512: return "SIMD-AVX512";
        case SimdISA::AVX2:   return "SIMD-AVX2";
        default:              return "SIMD-SSE42";
    }
}

std::vector<ClusterStats> SimdKMeans::cluster_stats(
    const float* data, size_t n, int dim) const {
    std::vector<int> labels;
    assign(data, n, dim, labels);
    return compute_cluster_stats(data, n, dim, labels.data(),
                                 centroids_.data(), k_);
}

}  // namespace eigenix
