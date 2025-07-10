pub mod vector;
pub mod ops;
pub mod matrix;
pub mod error;
pub use vector::Vector;
pub use ops::{F64Ops, F32Ops};
pub use matrix::Mat2D;
mod tests;