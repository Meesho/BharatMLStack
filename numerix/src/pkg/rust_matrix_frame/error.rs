use thiserror::Error;

#[derive(Debug, Error)]
pub enum InvalidOperation {
    #[error("Division by zero")]
    DivisionByZero,

    #[error("AND operation requires boolean values (0.0 or 1.0).")]
    AndRequiresBoolean,

    #[error("OR operation requires boolean values (0.0 or 1.0).")]
    OrRequiresBoolean,

    #[error("Cannot take logarithm of non-positive number")]
    LogNonPositive,

    #[error("p = q => Divided by 0")]
    PEqualsQDivByZero,

    #[error("Vector length cannot be 0 or less")]
    VectorLengthZeroOrLess,
}

#[derive(Debug, Error)]
pub enum Mat2DError {
    #[error("Matrix length mismatch: got {0}, expected {1} elements in matrix ({2}x{3})")]
    MatrixSizeMismatch(usize, usize, usize, usize),

    #[error("Column names count mismatch: expected {0} columns in the columns_name, but got {1}")]
    ColumnNamesMismatch(usize, usize),

    #[error("Stack underflow: missing operand for operation '{0}'")]
    StackUnderflow(String),

    #[error("Unknown token encountered: '{0}'")]
    UnknownToken(String),

    #[error("Evaluation error: stack is empty at the end of expression evaluation")]
    EmptyStack,

    #[error("Evaluation error: stack contains {0} elements at the end, expected exactly 1")]
    ExtraElementsInStack(usize),

    #[error("Invalid operation: {0}")]
    InvalidOperation(#[from] InvalidOperation),
}
