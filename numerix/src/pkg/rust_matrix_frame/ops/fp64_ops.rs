use super::VectorOps;
use crate::pkg::rust_matrix_frame::error::Mat2DError;
use crate::pkg::rust_matrix_frame::vector::Vector;
use std::f64;

pub struct F64Ops;

impl VectorOps for F64Ops {
    type Scalar = f64;

    fn new(size: usize) -> Vector<Self::Scalar> {
        Vector::new(size)
    }

    fn zeros(size: usize) -> Vector<Self::Scalar> {
        Vector::from_vec(vec![0.0; size])
    }

    fn add_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::new(left.len());

        for i in 0..left.len() {
            result[i] = left[i] + right[i];
        }

        result
    }

    fn sub_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::new(left.len());

        for i in 0..left.len() {
            result[i] = left[i] - right[i];
        }

        result
    }

    fn mul_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::new(left.len());

        for i in 0..left.len() {
            result[i] = left[i] * right[i];
        }

        result
    }

    fn div_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Result<Vector<Self::Scalar>, Mat2DError> {
        for i in 0..left.len() {
            if right[i] == 0.0 {
                return Err(Mat2DError::InvalidOperation("Division by zero".to_string()));
            }
        }

        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = left[i] / right[i];
        }
        Ok(result)
    }

    fn power_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = left[i].powf(right[i]);
        }
        result
    }

    fn min_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = left[i].min(right[i]);
        }
        result
    }

    fn max_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = left[i].max(right[i]);
        }
        result
    }

    fn greater_than_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = if left[i] > right[i] { 1.0 } else { 0.0 };
        }
        result
    }

    fn less_than_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = if left[i] < right[i] { 1.0 } else { 0.0 };
        }
        result
    }

    fn greater_than_equal_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = if left[i] >= right[i] { 1.0 } else { 0.0 };
        }
        result
    }

    fn less_than_equal_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = if left[i] <= right[i] { 1.0 } else { 0.0 };
        }
        result
    }

    fn equal_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            result[i] = if left[i] == right[i] { 1.0 } else { 0.0 };
        }
        result
    }

    fn and_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Result<Vector<Self::Scalar>, Mat2DError> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            if left[i] != 0.0 && left[i] != 1.0 || right[i] != 0.0 && right[i] != 1.0 {
                return Err(Mat2DError::InvalidOperation(
                    "AND operation requires boolean values (0.0 or 1.0)".to_string(),
                ));
            }
            result[i] = if left[i] != 0.0 && right[i] != 0.0 {
                1.0
            } else {
                0.0
            };
        }
        Ok(result)
    }

    fn or_vector(
        left: &Vector<Self::Scalar>,
        right: &Vector<Self::Scalar>,
    ) -> Result<Vector<Self::Scalar>, Mat2DError> {
        let mut result = Self::zeros(left.len());
        for i in 0..left.len() {
            if left[i] != 0.0 && left[i] != 1.0 || right[i] != 0.0 && right[i] != 1.0 {
                return Err(Mat2DError::InvalidOperation(
                    "OR operation requires boolean values (0.0 or 1.0)".to_string(),
                ));
            }
            result[i] = if left[i] != 0.0 || right[i] != 0.0 {
                1.0
            } else {
                0.0
            };
        }
        Ok(result)
    }

    fn log_vector(vec: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError> {
        for i in 0..vec.len() {
            if vec[i] <= 0.0 {
                return Err(Mat2DError::InvalidOperation(
                    "Cannot take logarithm of non-positive number".to_string(),
                ));
            }
        }

        let mut result = Self::zeros(vec.len());
        for i in 0..vec.len() {
            result[i] = vec[i].ln();
        }
        Ok(result)
    }

    fn abs_vector(vec: &Vector<Self::Scalar>) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(vec.len());
        for i in 0..vec.len() {
            result[i] = vec[i].abs();
        }
        result
    }

    fn exp_vector(vec: &Vector<Self::Scalar>) -> Vector<Self::Scalar> {
        let mut result = Self::zeros(vec.len());
        for i in 0..vec.len() {
            result[i] = vec[i].exp();
        }
        result
    }

    fn norm_min_max_vector(vec: &Vector<Self::Scalar>) -> Vector<Self::Scalar> {
        let mut min = f64::MAX;
        let mut max = f64::MIN;
        for i in 0..vec.len() {
            min = min.min(vec[i]);
            max = max.max(vec[i]);
        }
        Self::calculate_min_max_norm(vec, min, max)
    }

    fn norm_percentile_p_q(
        vec: &Vector<Self::Scalar>,
        p: Self::Scalar,
        q: Self::Scalar,
    ) -> Result<Vector<Self::Scalar>, Mat2DError> {
        if p == q {
            return Err(Mat2DError::InvalidOperation(
                "p = q => Divided by 0".to_string(),
            ));
        }
        if vec.len() == 1 {
            return Ok(Vector::from_vec(vec![1.0]));
        }
        let mut data = vec.as_slice().to_vec();
        data.sort_by(|a, b| a.partial_cmp(b).unwrap());
        let min = Self::compute_percentile(&data, p);
        let max = Self::compute_percentile(&data, q);
        if min == max {
            return Ok(Self::calculate_min_max_norm(vec, 1.0, 2.0));
        }
        let (min, max) = if min > max {
            let new_max = max + min;
            let new_min = new_max - min;
            let new_max = new_max - new_min;
            (new_min, new_max)
        } else {
            (min, max)
        };
        //println!("Min: {}, Max: {}", min, max);
        Ok(Self::calculate_min_max_norm(vec, min, max))
    }

    fn percentile_rank(vec: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError> {
        if vec.is_empty() {
            return Err(Mat2DError::InvalidOperation(
                "Vector length cannot be 0 or less".to_string(),
            ));
        }
        let mut result = Self::zeros(vec.len());
        let mut is_const_vector = true;

        for i in 0..vec.len() {
            result[i] = i as f64 / (vec.len() - 1) as f64;
            if i > 0 && vec[i] != vec[i - 1] {
                is_const_vector = false;
            }
        }

        if is_const_vector {
            for i in 0..vec.len() {
                result[i] = 1.0;
            }
        }

        Ok(result)
    }

    fn norm_percentile_0_99(
        vec: &Vector<Self::Scalar>,
    ) -> Result<Vector<Self::Scalar>, Mat2DError> {
        Self::norm_percentile_p_q(vec, 0.0, 99.0)
    }

    fn norm_percentile_5_95(
        vec: &Vector<Self::Scalar>,
    ) -> Result<Vector<Self::Scalar>, Mat2DError> {
        Self::norm_percentile_p_q(vec, 5.0, 95.0)
    }
}

impl F64Ops {
    fn calculate_min_max_norm(vec: &Vector<f64>, min: f64, max: f64) -> Vector<f64> {
        let mut result = Vector::new(vec.len());
        let delta = max - min;
        if delta != 0.0 {
            for i in 0..vec.len() {
                result[i] = (vec[i] - min) / delta;
            }
        } else {
            for i in 0..vec.len() {
                result[i] = 1.0;
            }
        }
        //println!("Result: {:?}", result);
        result
    }

    fn compute_percentile(values: &[f64], p: f64) -> f64 {
        if p == 0.0 {
            return values[0];
        }
        let index = p / 100.0 * (values.len() - 1) as f64;
        let floor = index.floor();
        if floor == (values.len() - 1) as f64 {
            return values[values.len() - 1];
        }
        let diff = index - floor;
        if diff == 0.0 {
            return values[floor as usize];
        }
        //println!("Floor: {}, Diff: {}", floor, diff);
        values[floor as usize] + (values[floor as usize + 1] - values[floor as usize]) * diff
    }
}
