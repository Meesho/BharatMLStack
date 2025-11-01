// crates/kvr-io/benches/io.rs
use criterion::{criterion_group, criterion_main, Criterion, Throughput, BenchmarkId, black_box};
use kvr_io::{Io, PreadIo};
use rand::{rngs::StdRng, Rng, SeedableRng};
use std::{fs::File, io::Write, time::{Duration, Instant}};

const BLOCK: usize = 8192;
const FILE_SIZE: u64 = 1 * 1024 * 1024 * 1024; // 1 GiB
const N_BLOCKS: u64 = FILE_SIZE / BLOCK as u64;

fn make_file(path: &str) {
    let mut f = File::create(path).unwrap();
    let buf = vec![0u8; BLOCK];
    for _ in 0..N_BLOCKS {
        f.write_all(&buf).unwrap();
    }
    f.sync_all().unwrap();
}

fn bench_pread_single(c: &mut Criterion) {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("data.bin");
    make_file(path.to_str().unwrap());
    let io = PreadIo::open(path.to_str().unwrap(), false).unwrap();

    // Reuse one buffer
    let mut buf = vec![0u8; BLOCK];

    let mut group = c.benchmark_group("pread_single_8k");
    group.throughput(Throughput::Bytes(BLOCK as u64));

    group.bench_function(BenchmarkId::from_parameter("warm-pagecache"), |b| {
        // Measure many ops per sample to increase resolution
        b.iter_custom(|iters| {
            let mut rng = StdRng::seed_from_u64(42);
            let start = Instant::now();
            for _ in 0..iters {
                // random block offset
                let bid = rng.gen_range(0..N_BLOCKS) as u64;
                let off = bid * BLOCK as u64;
                io.read_at(&mut buf[..], off).unwrap();
                black_box(buf[0]); // prevent optimization
            }
            start.elapsed()
        });
    });

    group.finish();
}

fn bench_pread_batch_32(c: &mut Criterion) {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("data.bin");
    make_file(path.to_str().unwrap());
    let io = PreadIo::open(path.to_str().unwrap(), false).unwrap();

    // Reuse 32 buffers
    let mut bufs: Vec<Vec<u8>> = (0..32).map(|_| vec![0u8; BLOCK]).collect();

    let mut group = c.benchmark_group("pread_batch_32x8k");
    group.throughput(Throughput::Bytes((32 * BLOCK) as u64));

    group.bench_function(BenchmarkId::from_parameter("looped"), |b| {
        b.iter_custom(|iters| {
            let mut rng = StdRng::seed_from_u64(123);
            let start = Instant::now();
            for _ in 0..iters {
                for dst in &mut bufs {
                    let bid = rng.gen_range(0..N_BLOCKS) as u64;
                    let off = bid * BLOCK as u64;
                    io.read_at(&mut dst[..], off).unwrap();
                    black_box(dst[0]);
                }
            }
            start.elapsed()
        });
    });

    group.finish();
}


fn bench_pread_single_odirect(c: &mut Criterion) {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("data.bin");
    make_file(path.to_str().unwrap());

    // Linux: O_DIRECT=true, 4k-aligned offsets/lengths already satisfied
    let io = PreadIo::open(path.to_str().unwrap(), true).unwrap();
    let mut buf = vec![0u8; BLOCK];

    let mut group = c.benchmark_group("pread_single_8k_odirect");
    group.bench_function("qd1", |b| {
        b.iter_custom(|iters| {
            let mut rng = StdRng::seed_from_u64(99);
            let start = Instant::now();
            for _ in 0..iters {
                let bid = rng.gen_range(0..N_BLOCKS);
                let off = bid * BLOCK as u64; // 4k aligned
                io.read_at(&mut buf[..], off).unwrap();
                black_box(buf[0]);
            }
            start.elapsed()
        });
    });
    group.finish();
}

fn bench_pread_batch32_odirect(c: &mut Criterion) {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("data.bin");
    make_file(path.to_str().unwrap());

    let io = PreadIo::open(path.to_str().unwrap(), true).unwrap();
    let mut bufs: Vec<Vec<u8>> = (0..32).map(|_| vec![0u8; BLOCK]).collect();

    let mut group = c.benchmark_group("pread_batch_32x8k_odirect");
    group.bench_function("qd~32_looped", |b| {
        b.iter_custom(|iters| {
            let mut rng = StdRng::seed_from_u64(1234);
            let start = Instant::now();
            for _ in 0..iters {
                for dst in &mut bufs {
                    let bid = rng.gen_range(0..N_BLOCKS);
                    let off = bid * BLOCK as u64;
                    io.read_at(&mut dst[..], off).unwrap();
                    black_box(dst[0]);
                }
            }
            start.elapsed()
        });
    });
    group.finish();
}

criterion_group! {
    name = benches;
    config = Criterion::default()
        .measurement_time(Duration::from_secs(6))
        .warm_up_time(Duration::from_secs(2))
        .sample_size(40);
    targets = bench_pread_single, bench_pread_batch_32
}
criterion_main!(benches);
