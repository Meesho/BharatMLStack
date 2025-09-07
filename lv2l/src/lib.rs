pub mod pq;

pub use pq::*;

#[cfg(test)]
mod tests {
    use super::*;
    use ndarray::Array;
    use rand::prelude::*;

    #[test]
    fn test_product_quantization() {
        // Create sample data
        let mut rng = thread_rng();
        let n_vectors = 100;
        let total_dims = 16 * 50; // 16 blocks of 50 dimensions
        
        let data = Array::from_shape_fn((n_vectors, total_dims), |_| rng.gen_range(0.0..1.0));
        
        // Build PQ index
        let pq_result = ProductQuantizer::build_level0(data.view(), 16, 50, 16, false);
        assert!(pq_result.is_ok());
        
        let pq = pq_result.unwrap();
        println!("Built PQ index with {} vectors", pq.codes.len());
        
        // Test encoding/decoding
        let test_vec = data.row(0);
        let encoded = pq.encode_vector(test_vec).unwrap();
        let decoded = pq.decode_code(encoded);
        
        assert_eq!(decoded.len(), total_dims);
        
        // Print stats
        let stats = pq.stats();
        println!("PQ Statistics: {:#?}", stats);
    }
}
