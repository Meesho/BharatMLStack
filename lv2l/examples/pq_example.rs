use lv2l::ProductQuantizer;
use ndarray::Array;
use rand::prelude::*;

fn main() -> Result<(), String> {
    println!("Product Quantization Example");
    
    // Generate synthetic data
    let mut rng = thread_rng();
    let n_vectors = 10000;
    let total_dims = 16 * 50; // 16 blocks of 50 dimensions each
    
    println!("Generating {} vectors of {} dimensions...", n_vectors, total_dims);
    let data = Array::from_shape_fn((n_vectors, total_dims), |_| rng.gen_range(0.0..1.0));
    
    // Build Product Quantization index
    println!("Building PQ index...");
    let pq = ProductQuantizer::build_level0(
        data.view(),
        16,  // M = 16 blocks
        50,  // m = 50 dimensions per block  
        16,  // k = 16 centroids per block
        false // use_opq = false (no rotation)
    )?;
    
    println!("PQ index built successfully!");
    
    // Print statistics
    let stats = pq.stats();
    println!("\nPQ Index Statistics:");
    println!("- Total vectors: {}", stats.total_vectors);
    println!("- Total blocks: {}", stats.total_blocks);
    println!("- Dimensions per block: {}", stats.dims_per_block);
    println!("- Centroids per block: {}", stats.centroids_per_block);
    println!("- Average bucket size: {:.2}", stats.avg_bucket_size);
    println!("- Max bucket size: {}", stats.max_bucket_size);
    println!("- Min bucket size: {}", stats.min_bucket_size);
    println!("- Empty buckets: {}", stats.empty_buckets);
    println!("- Total buckets: {}", stats.total_buckets);
    
    // Test encoding and decoding
    println!("\nTesting encoding/decoding...");
    let test_vector = data.row(0);
    let original_code = pq.codes[0];
    let encoded_code = pq.encode_vector(test_vector)?;
    
    println!("Original code: 0x{:016x}", original_code);
    println!("Encoded code:  0x{:016x}", encoded_code);
    println!("Codes match: {}", original_code == encoded_code);
    
    // Decode the vector
    let decoded_vector = pq.decode_code(encoded_code);
    
    // Calculate reconstruction error
    let diff = &test_vector.to_owned() - &decoded_vector;
    let mse = diff.dot(&diff) / decoded_vector.len() as f32;
    println!("Mean Squared Error (reconstruction): {:.6}", mse);
    
    // Test bucket access
    println!("\nTesting bucket access...");
    for block_id in 0..3 { // Check first 3 blocks
        for centroid_id in 0..3 { // Check first 3 centroids
            if let Some(bucket) = pq.get_bucket(block_id, centroid_id) {
                println!("Block {}, Centroid {}: {} vectors", 
                    block_id, centroid_id, bucket.len());
                if !bucket.is_empty() {
                    println!("  First few vector IDs: {:?}", 
                        &bucket[..bucket.len().min(5)]);
                }
            }
        }
    }
    
    // Demonstrate compression ratio
    let original_size = n_vectors * total_dims * 4; // 4 bytes per f32
    let compressed_size = n_vectors * 8 + // 8 bytes per code (u64)
                         pq.codebooks.iter().map(|cb| cb.len() * 4).sum::<usize>(); // codebook size
    let compression_ratio = original_size as f32 / compressed_size as f32;
    
    println!("\nCompression Analysis:");
    println!("- Original size: {} bytes ({:.2} MB)", original_size, original_size as f32 / 1_000_000.0);
    println!("- Compressed size: {} bytes ({:.2} MB)", compressed_size, compressed_size as f32 / 1_000_000.0);
    println!("- Compression ratio: {:.2}x", compression_ratio);
    
    Ok(())
} 