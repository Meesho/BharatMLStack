use ndarray::{Array1, Array2, ArrayView1, ArrayView2, s};
use rand::prelude::*;
use rayon::prelude::*;
use serde::{Deserialize, Serialize};

/// Product Quantization Index
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProductQuantizer {
    /// Number of blocks (M)
    pub m_blocks: usize,
    /// Dimensions per block (m)
    pub dims_per_block: usize,
    /// Number of centroids per block (k)
    pub k_centroids: usize,
    /// Codebooks: M blocks, each with k centroids of dims_per_block dimensions
    pub codebooks: Vec<Array2<f32>>,
    /// Buckets: [block][centroid] -> list of vector IDs
    pub buckets: Vec<Vec<Vec<usize>>>,
    /// Codes for each vector (packed as u64)
    pub codes: Vec<u64>,
    /// Optional rotation matrix P for OPQ
    pub rotation_matrix: Option<Array2<f32>>,
}

/// Buffer for collecting IDs before sealing
#[derive(Debug, Clone)]
pub struct IDBuffer {
    ids: Vec<usize>,
    sealed: bool,
}

impl IDBuffer {
    pub fn new() -> Self {
        Self {
            ids: Vec::new(),
            sealed: false,
        }
    }

    pub fn append(&mut self, id: usize) {
        if !self.sealed {
            self.ids.push(id);
        }
    }

    pub fn seal_and_flush(&mut self) -> Vec<usize> {
        self.sealed = true;
        std::mem::take(&mut self.ids)
    }
}

impl ProductQuantizer {
    /// Build Level 0 Product Quantization Index
    /// 
    /// # Arguments
    /// * `x` - Input dataset of shape (N, D)
    /// * `m_blocks` - Number of blocks (M), default 16
    /// * `dims_per_block` - Dimensions per block (m), default 50  
    /// * `k_centroids` - Number of centroids per block (k), default 16
    /// * `use_opq` - Whether to use OPQ rotation (optional)
    pub fn build_level0(
        x: ArrayView2<f32>,
        m_blocks: usize,
        dims_per_block: usize,
        k_centroids: usize,
        use_opq: bool,
    ) -> Result<Self, String> {
        let (n_vectors, total_dims) = x.dim();
        
        // Verify dimensions
        if total_dims != m_blocks * dims_per_block {
            return Err(format!(
                "Dimension mismatch: {} != {} * {}",
                total_dims, m_blocks, dims_per_block
            ));
        }

        // Step 0: Optional rotation (OPQ or identity)
        let (rotation_matrix, x_rotated) = if use_opq {
            let p = Self::train_opq(&x, m_blocks, dims_per_block)?;
            let x_rot = x.dot(&p);
            (Some(p), x_rot)
        } else {
            (None, x.to_owned())
        };

        // Step 1: Learn k-means codebooks per block (parallel)
        println!("Learning codebooks for {} blocks...", m_blocks);
        let codebooks_results: Vec<_> = (0..m_blocks)
            .into_par_iter()
            .map(|j| {
                let start_col = j * dims_per_block;
                let end_col = (j + 1) * dims_per_block;
                let x_block = x_rotated.slice(s![.., start_col..end_col]);
                Self::kmeans(&x_block, k_centroids, 100) // 100 max iterations
            })
            .collect();
        
        let mut codebooks = Vec::new();
        for result in codebooks_results {
            codebooks.push(result?);
        }

        // Step 2: Allocate buckets
        let mut buckets: Vec<Vec<IDBuffer>> = (0..m_blocks)
            .map(|_| (0..k_centroids).map(|_| IDBuffer::new()).collect())
            .collect();

        // Step 3: Encode vectors & fill buckets
        println!("Encoding {} vectors...", n_vectors);
        let mut codes = vec![0u64; n_vectors];
        
        for (id, x_vec) in x_rotated.rows().into_iter().enumerate() {
            let mut code64 = 0u64;
            
            for j in 0..m_blocks {
                let start_idx = j * dims_per_block;
                let end_idx = (j + 1) * dims_per_block;
                let slice = x_vec.slice(s![start_idx..end_idx]);
                
                // Find closest centroid
                let cid = Self::argmin_l2_distance(&slice, &codebooks[j])?;
                
                // Add to bucket
                buckets[j][cid].append(id);
                
                // Pack 4 bits into code64
                code64 |= (cid as u64) << (4 * j);
            }
            
            codes[id] = code64;
        }

        // Step 4: Seal buckets (convert to contiguous memory)
        let sealed_buckets: Vec<Vec<Vec<usize>>> = buckets
            .into_iter()
            .map(|block_buckets| {
                block_buckets
                    .into_iter()
                    .map(|mut bucket| bucket.seal_and_flush())
                    .collect()
            })
            .collect();

        Ok(ProductQuantizer {
            m_blocks,
            dims_per_block,
            k_centroids,
            codebooks,
            buckets: sealed_buckets,
            codes,
            rotation_matrix,
        })
    }

    /// Simple OPQ training (placeholder - returns identity matrix)
    /// In practice, this would involve iterative optimization
    fn train_opq(
        _x: &ArrayView2<f32>,
        _m_blocks: usize,
        dims_per_block: usize,
    ) -> Result<Array2<f32>, String> {
        let total_dims = _m_blocks * dims_per_block;
        Ok(Array2::eye(total_dims))
    }

    /// K-means clustering implementation
    fn kmeans(
        x: &ArrayView2<f32>,
        k: usize,
        max_iterations: usize,
    ) -> Result<Array2<f32>, String> {
        let (n_samples, n_features) = x.dim();
        
        if k > n_samples {
            return Err("k cannot be larger than number of samples".to_string());
        }

        let mut rng = thread_rng();
        
        // Initialize centroids randomly
        let mut centroids = Array2::<f32>::zeros((k, n_features));
        for i in 0..k {
            let random_idx = rng.gen_range(0..n_samples);
            centroids.row_mut(i).assign(&x.row(random_idx));
        }

        let mut assignments = vec![0usize; n_samples];
        
        for _iteration in 0..max_iterations {
            let mut changed = false;
            
            // Assignment step
            for (i, sample) in x.rows().into_iter().enumerate() {
                let new_assignment = Self::argmin_l2_distance(&sample, &centroids)?;
                if assignments[i] != new_assignment {
                    assignments[i] = new_assignment;
                    changed = true;
                }
            }
            
            // Update step
            for cluster_id in 0..k {
                let cluster_samples: Vec<_> = assignments
                    .iter()
                    .enumerate()
                    .filter(|&(_, assignment)| *assignment == cluster_id)
                    .map(|(i, _)| x.row(i))
                    .collect();
                
                if !cluster_samples.is_empty() {
                    let mut centroid = Array1::<f32>::zeros(n_features);
                    for sample in &cluster_samples {
                        centroid = centroid + sample;
                    }
                    centroid = centroid / cluster_samples.len() as f32;
                    centroids.row_mut(cluster_id).assign(&centroid);
                }
            }
            
            if !changed {
                break;
            }
        }

        Ok(centroids)
    }

    /// Find the index of the closest centroid using L2 distance
    fn argmin_l2_distance(
        point: &ArrayView1<f32>,
        centroids: &Array2<f32>,
    ) -> Result<usize, String> {
        let mut min_distance = f32::INFINITY;
        let mut closest_idx = 0;

        for (i, centroid) in centroids.rows().into_iter().enumerate() {
            let diff = point - &centroid;
            let distance = diff.dot(&diff); // L2 squared distance
            
            if distance < min_distance {
                min_distance = distance;
                closest_idx = i;
            }
        }

        Ok(closest_idx)
    }

    /// Encode a single vector using the trained codebooks
    pub fn encode_vector(&self, x: ArrayView1<f32>) -> Result<u64, String> {
        let x_rotated = if let Some(ref p) = self.rotation_matrix {
            p.t().dot(&x)
        } else {
            x.to_owned()
        };

        let mut code64 = 0u64;
        
        for j in 0..self.m_blocks {
            let start_idx = j * self.dims_per_block;
            let end_idx = (j + 1) * self.dims_per_block;
            let slice = x_rotated.slice(s![start_idx..end_idx]);
            
            let cid = Self::argmin_l2_distance(&slice, &self.codebooks[j])?;
            code64 |= (cid as u64) << (4 * j);
        }

        Ok(code64)
    }

    /// Decode a code back to approximate vector
    pub fn decode_code(&self, code: u64) -> Array1<f32> {
        let mut decoded = Array1::<f32>::zeros(self.m_blocks * self.dims_per_block);
        
        for j in 0..self.m_blocks {
            let cid = ((code >> (4 * j)) & 0xF) as usize;
            let start_idx = j * self.dims_per_block;
            let end_idx = (j + 1) * self.dims_per_block;
            
            let centroid = self.codebooks[j].row(cid);
            decoded.slice_mut(s![start_idx..end_idx]).assign(&centroid);
        }

        // Apply inverse rotation if OPQ was used
        if let Some(ref p) = self.rotation_matrix {
            p.dot(&decoded)
        } else {
            decoded
        }
    }

    /// Get vectors in a specific bucket
    pub fn get_bucket(&self, block_id: usize, centroid_id: usize) -> Option<&Vec<usize>> {
        self.buckets.get(block_id)?.get(centroid_id)
    }

    /// Get statistics about the PQ index
    pub fn stats(&self) -> PQStats {
        let total_vectors = self.codes.len();
        let mut bucket_sizes = Vec::new();
        let mut empty_buckets = 0;

        for block in &self.buckets {
            for bucket in block {
                bucket_sizes.push(bucket.len());
                if bucket.is_empty() {
                    empty_buckets += 1;
                }
            }
        }

        let avg_bucket_size = bucket_sizes.iter().sum::<usize>() as f32 / bucket_sizes.len() as f32;
        let max_bucket_size = *bucket_sizes.iter().max().unwrap_or(&0);
        let min_bucket_size = *bucket_sizes.iter().min().unwrap_or(&0);

        PQStats {
            total_vectors,
            total_blocks: self.m_blocks,
            dims_per_block: self.dims_per_block,
            centroids_per_block: self.k_centroids,
            avg_bucket_size,
            max_bucket_size,
            min_bucket_size,
            empty_buckets,
            total_buckets: self.m_blocks * self.k_centroids,
        }
    }
}

#[derive(Debug)]
pub struct PQStats {
    pub total_vectors: usize,
    pub total_blocks: usize,
    pub dims_per_block: usize,
    pub centroids_per_block: usize,
    pub avg_bucket_size: f32,
    pub max_bucket_size: usize,
    pub min_bucket_size: usize,
    pub empty_buckets: usize,
    pub total_buckets: usize,
}

#[cfg(test)]
mod tests {
    use super::*;
    use ndarray::Array;

    #[test]
    fn test_pq_build_and_encode() {
        // Create synthetic data: 1000 vectors of 800 dimensions (16 blocks * 50 dims)
        let mut rng = thread_rng();
        let n_vectors = 1000;
        let total_dims = 16 * 50; // M=16, m=50
        
        let data = Array::from_shape_fn((n_vectors, total_dims), |_| rng.gen_range(0.0..1.0));
        
        // Build PQ index
        let pq = ProductQuantizer::build_level0(data.view(), 16, 50, 16, false).unwrap();
        
        // Check dimensions
        assert_eq!(pq.m_blocks, 16);
        assert_eq!(pq.dims_per_block, 50);
        assert_eq!(pq.k_centroids, 16);
        assert_eq!(pq.codes.len(), n_vectors);
        
        // Test encoding a vector
        let test_vector = data.row(0);
        let code = pq.encode_vector(test_vector).unwrap();
        assert_eq!(code, pq.codes[0]);
        
        // Test decoding
        let decoded = pq.decode_code(code);
        assert_eq!(decoded.len(), total_dims);
        
        // Print stats
        let stats = pq.stats();
        println!("PQ Stats: {:#?}", stats);
    }
}
