use std::collections::HashMap;
use std::str::FromStr;
use std::fmt::{Debug, Display};
use crate::ops::VectorOps;
use crate::vector::Vector;
use crate::error::Mat2DError;
use std::borrow::Cow;   

type BinaryOp<V> = fn(&Vector<<V as VectorOps>::Scalar>, &Vector<<V as VectorOps>::Scalar>) -> Vector<<V as VectorOps>::Scalar>;
type BinaryOpResult<V> = fn(&Vector<<V as VectorOps>::Scalar>, &Vector<<V as VectorOps>::Scalar>) -> Result<Vector<<V as VectorOps>::Scalar>, Mat2DError>;
type UnaryOp<V> = fn(&Vector<<V as VectorOps>::Scalar>) -> Vector<<V as VectorOps>::Scalar>;
type UnaryOpResult<V> = fn(&Vector<<V as VectorOps>::Scalar>) -> Result<Vector<<V as VectorOps>::Scalar>, Mat2DError>;

enum Operation<V: VectorOps> {
    Binary(BinaryOp<V>),
    BinaryResult(BinaryOpResult<V>), 
    Unary(UnaryOp<V>),
    UnaryResult(UnaryOpResult<V>),
}

impl<V: VectorOps> Operation<V> {
    fn get_binary_ops() -> HashMap<&'static str, Operation<V>> {
        let mut m = HashMap::new();
        // Basic arithmetic
        m.insert("+", Operation::Binary(V::add_vector));
        m.insert("-", Operation::Binary(V::sub_vector));
        m.insert("*", Operation::Binary(V::mul_vector));
        m.insert("/", Operation::BinaryResult(V::div_vector));
        m.insert("^", Operation::Binary(V::power_vector));
        
        // Comparison operations
        m.insert(">", Operation::Binary(V::greater_than_vector));
        m.insert("<", Operation::Binary(V::less_than_vector));
        m.insert(">=", Operation::Binary(V::greater_than_equal_vector));
        m.insert("<=", Operation::Binary(V::less_than_equal_vector));
        m.insert("==", Operation::Binary(V::equal_vector));
        
        // Min/Max operations
        m.insert("min", Operation::Binary(V::min_vector));
        m.insert("max", Operation::Binary(V::max_vector));
        
        // Boolean operations
        m.insert("&", Operation::BinaryResult(V::and_vector));
        m.insert("|", Operation::BinaryResult(V::or_vector));
        
        m
    }

    fn get_unary_ops() -> HashMap<&'static str, Operation<V>> {
        let mut m = HashMap::new();
        // Mathematical functions
        m.insert("exp", Operation::Unary(V::exp_vector));
        m.insert("log", Operation::UnaryResult(V::log_vector));
        m.insert("abs", Operation::Unary(V::abs_vector));
        
        // Normalization operations
        m.insert("norm_min_max", Operation::Unary(V::norm_min_max_vector));
        m.insert("percentile_rank", Operation::UnaryResult(V::percentile_rank));
        m.insert("norm_percentile_0_99", Operation::UnaryResult(V::norm_percentile_0_99));
        m.insert("norm_percentile_5_95", Operation::UnaryResult(V::norm_percentile_5_95));
        m
    }
} 

#[derive(Debug, Clone)]
pub struct Mat2D<V: VectorOps> {
    pub rows: usize, 
    pub cols: usize, 
    pub matrix: Vec<V::Scalar>,
    pub column_names : HashMap<String, usize>,
}

impl<V: VectorOps> Mat2D<V>
where 
    V::Scalar : Copy + PartialOrd + Default, 
{
    pub fn from_data(rows: usize, cols: usize, matrix: Vec<V::Scalar>, column_names: HashMap<String, usize>) -> Result<Self, Mat2DError>{
       if rows*cols != matrix.len() {
            return Err(Mat2DError::MatrixSizeMismatch(matrix.len(), rows * cols, rows, cols));
       } 

       if cols != column_names.len() {
            return Err(Mat2DError::ColumnNamesMismatch(cols, column_names.len()));
       } 

       Ok(Mat2D {
            rows, 
            cols, 
            matrix, 
            column_names,
        })
        
    }

    fn get_column(&self, name: &str) -> Option<Vector<V::Scalar>>{
        self.column_names.get(name).map(|&col_ind|{
            let mut vec = Vector::new(self.rows);
            for i in 0..self.rows {
                vec[i] = self.matrix[col_ind * self.rows + i];
            }
            vec
        })
    }

    pub fn calculate(&self, expr: &str, meta: HashMap<String, Vector<V::Scalar>>) -> Result<Vector<V::Scalar>, Mat2DError > 
    where
        V::Scalar: FromStr + Display + Default,
        <V::Scalar as FromStr>::Err: Debug,
    {
        let mut stack: Vec<Cow<'_, Vector<V::Scalar>>> = Vec::new();
        let tokens : Vec<&str> = expr.split_whitespace().collect();

        let binary_ops_list = Operation::<V>::get_binary_ops();
        let unary_ops_list = Operation::<V>::get_unary_ops();

        for token in tokens {
            match (
                binary_ops_list.get(token),
                unary_ops_list.get(token),
                meta.get(token),
                self.get_column(token),
            ) {
                (Some(op), _, _, _) => {
                    let right = stack.pop().ok_or_else(|| Mat2DError::StackUnderflow(token.to_string()))?;
                    let left = stack.pop().ok_or_else(|| Mat2DError::StackUnderflow(token.to_string()))?;

                    let result = match op {
                        Operation::Binary(func) => func(&left, &right),
                        Operation::BinaryResult(func) => func(&left, &right)?,
                        _ => unreachable!(),
                    };
                    stack.push(Cow::Owned(result));
                }

                // Unary operation
                (_, Some(op), _, _) => {
                    let vec = stack.pop().ok_or_else(|| Mat2DError::StackUnderflow(token.to_string()))?;
                
                    let result = match op {
                        Operation::Unary(func) => func(&vec),
                        Operation::UnaryResult(func) => func(&vec)?,
                        _ => unreachable!(),
                    };
                    stack.push(Cow::Owned(result));
                }

                (_, _, Some(static_vector), _) => {
                    stack.push(Cow::Borrowed(static_vector));
                }

                // Column vector from `self`
                (_, _, _, Some(vector)) => {
                    stack.push(Cow::Owned(vector));
                }
                
                _ => return Err(Mat2DError::UnknownToken(token.to_string())),

            }
        }
        if stack.len() > 1 {
            return Err(Mat2DError::ExtraElementsInStack(stack.len()));
        }
        stack.pop().ok_or_else(|| Mat2DError::EmptyStack).map(|cow| cow.into_owned())
    }
}
