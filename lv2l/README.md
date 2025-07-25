# LV2L - Product Quantization Library

A Rust implementation of Product Quantization (PQ) for efficient vector compression and approximate nearest neighbor search.

## Overview

Product Quantization is a technique for compressing high-dimensional vectors while preserving approximate distances. This implementation follows the `BUILD_LEVEL0` algorithm and includes:

- **Product Quantization**: Divide vectors into blocks and quantize each block independently
- **Parallel K-means**: Learn codebooks for each block in parallel using Rayon
- **Bucket Organization**: Efficient storage and retrieval of quantized vectors
- **Optional OPQ**: Support for Optimized Product Quantization (rotation matrix)
- **Serialization**: Save/load PQ indices using Serde

## Algorithm

The implementation follows this procedure:

```
procedure BUILD_LEVEL0(X, M=16, m=50, k=16):
    # 0. Optional rotation
    P    ← train_OPQ(X, M, m)            # or identity if skipped
    Xrot ← [ P · x  for x in X ]

    # 1. Learn k-means code-books per block
    for j in 0..M-1 parallel:
        Xj    ← Xrot[:, m*j : m*(j+1)]          # shape (N, 50)
        C[j]  ← kmeans(Xj, k)                   # 16 centroids of 50-D

    # 2. Allocate buckets
    for j in 0..M-1:
        for c in 0..k-1:
            B[j][c] ← new IDBuffer()

    # 3. Encode vectors & fill buckets
    for id, x in enumerate(Xrot):
        code64 ← 0
        for j in 0..M-1:
            slice  ← x[m*j : m*(j+1)]
            cid    ← argmin_t  L2(slice, C[j][t])      # 0..15
            B[j][cid].append(id)
            code64 |= (cid << (4*j))                   # pack 4 bits
        Codes[id] = code64

    # 4. Seal buckets (contiguous memory) and persist
    for j,c:
        B[j][c].seal_and_flush()

    return {C, B, Codes, P}
```

## Features

- **Efficient Storage**: Each vector is compressed to a 64-bit code
- **Fast Encoding**: O(M*k) distance computations per vector
- **Bucket Access**: Quick retrieval of vectors by quantization codes
- **Statistics**: Comprehensive metrics about the PQ index
- **Parallel Training**: K-means runs in parallel across blocks
- **Memory Efficient**: Contiguous memory layout for buckets

## Usage

### Basic Example

```rust
use lv2l::ProductQuantizer;
use ndarray::Array;
use rand::prelude::*;

// Generate sample data
let mut rng = thread_rng();
let n_vectors = 10000;
let total_dims = 16 * 50; // 16 blocks of 50 dimensions
let data = Array::from_shape_fn((n_vectors, total_dims), |_| rng.gen::<f32>());

// Build PQ index
let pq = ProductQuantizer::build_level0(
    data.view(),
    16,    // M = 16 blocks
    50,    // m = 50 dimensions per block
    16,    // k = 16 centroids per block
    false  // use_opq = false (no rotation)
)?;

// Encode a vector
let test_vector = data.row(0);
let code = pq.encode_vector(test_vector)?;

// Decode back to approximate vector
let decoded = pq.decode_code(code);

// Access buckets
if let Some(bucket) = pq.get_bucket(0, 5) {
    println!("Block 0, Centroid 5 contains {} vectors", bucket.len());
}

// Get statistics
let stats = pq.stats();
println!("Compression ratio: {:.2}x", 
    (n_vectors * total_dims * 4) as f32 / 
    (n_vectors * 8 + pq.codebooks.iter().map(|cb| cb.len() * 4).sum::<usize>()) as f32
);
```

### Running the Example

```bash
cargo run --example pq_example
```

### Running Tests

```bash
cargo test
```

## Data Structures

### ProductQuantizer

The main PQ index containing:
- `codebooks`: M blocks of k centroids each
- `buckets`: Organized storage of vector IDs by quantization codes
- `codes`: 64-bit compressed representation of each vector
- `rotation_matrix`: Optional OPQ rotation matrix

### IDBuffer

Temporary buffer for collecting vector IDs during index construction, later sealed into contiguous memory.

### PQStats

Statistics about the PQ index including compression ratios, bucket distributions, and memory usage.

## Parameters

- **M (m_blocks)**: Number of blocks to divide vectors into (default: 16)
- **m (dims_per_block)**: Dimensions per block (default: 50)
- **k (k_centroids)**: Number of centroids per block (default: 16)
- **use_opq**: Whether to use Optimized Product Quantization rotation

## Memory Layout

- **Original vectors**: N × D × 4 bytes (f32)
- **Compressed codes**: N × 8 bytes (u64)
- **Codebooks**: M × k × m × 4 bytes (f32)
- **Buckets**: Variable size based on distribution

Typical compression ratio: **10-50x** depending on parameters.

## Dependencies

- `ndarray`: Matrix operations and linear algebra
- `rayon`: Parallel processing for k-means
- `rand`: Random number generation for k-means initialization
- `serde`: Serialization support

## Limitations

- Current OPQ implementation returns identity matrix (placeholder)
- K-means uses simple initialization (random centroids)
- Maximum 16 centroids per block (4-bit codes)
- Fixed 64-bit code length

## Future Improvements

- Advanced OPQ optimization
- Better k-means initialization (k-means++)
- Variable-length codes
- GPU acceleration
- Advanced distance metrics 