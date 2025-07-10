use crate::vector::Vector;
use serde::{Serialize, Deserialize};
use crate::error::Mat2DError;
mod fp32_ops;
mod fp64_ops;

pub use fp32_ops::F32Ops;
pub use fp64_ops::F64Ops;

pub trait VectorOps {
    type Scalar: Copy + PartialOrd + Serialize + for<'de> Deserialize<'de>;
    fn zeros(size: usize) -> Vector<Self::Scalar>;
    fn new(size: usize) -> Vector<Self::Scalar>;
    fn add_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn sub_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn mul_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn div_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError>;
    fn power_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn min_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn max_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn greater_than_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn less_than_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn greater_than_equal_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn less_than_equal_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn equal_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn and_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError>;
    fn or_vector(left: &Vector<Self::Scalar>, right: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError>;
    fn log_vector(left: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError>;
    fn abs_vector(left: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn exp_vector(left: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn norm_min_max_vector(left: &Vector<Self::Scalar>) -> Vector<Self::Scalar>;
    fn norm_percentile_p_q(left: &Vector<Self::Scalar>, p: Self::Scalar, q: Self::Scalar) -> Result<Vector<Self::Scalar>, Mat2DError>;
    fn percentile_rank(left: &Vector<Self::Scalar>) ->  Result<Vector<Self::Scalar>, Mat2DError>;
    fn norm_percentile_0_99(vec: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError>;
    fn norm_percentile_5_95(vec: &Vector<Self::Scalar>) -> Result<Vector<Self::Scalar>, Mat2DError>;
}

