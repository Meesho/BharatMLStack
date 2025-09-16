use crate::pkg::rust_matrix_frame::matrix::Mat2D;
use crate::pkg::rust_matrix_frame::ops::F32Ops;
use crate::pkg::rust_matrix_frame::vector::Vector;
use std::collections::HashMap;

fn setup() -> Mat2D<F32Ops> {
    let mut column_names = HashMap::new();
    column_names.insert("pctr".to_string(), 0);
    column_names.insert("pcvr".to_string(), 1);
    column_names.insert("nobygo".to_string(), 2);
    column_names.insert("x".to_string(), 3);
    column_names.insert("y".to_string(), 4);
    column_names.insert("z".to_string(), 5);
    column_names.insert("a".to_string(), 6);

    let matrix = vec![
        0.1, 0.0, 0.1, 0.0, // Column 1
        0.2, 0.1, 0.2, 0.5, // Column 2
        -0.3, 0.2, -0.5, 0.1, // Column 3
        0.6, 0.6, 0.6, 0.6, // Column 4
        0.2, 1.0, 0.0, 0.6, // Column 5
        0.0, 0.0, 1.0, 1.0, // Column 6
        0.0, 1.0, 0.0, 1.0, // Column 7
    ];

    Mat2D::from_data(4, 7, matrix, column_names).unwrap()
}

fn vectors_are_equal(vec1: &Vector<f32>, vec2: &[f32]) -> bool {
    if vec1.len() != vec2.len() {
        return false;
    }
    for i in 0..vec1.len() {
        if (vec1[i] - vec2[i]).abs() > 10e-7 {
            return false;
        }
    }
    true
}

#[test]
fn test_add_vectors() {
    let mat = setup();
    let result = mat.calculate("pctr pcvr +", HashMap::new()).unwrap();
    let expected = vec![0.3, 0.1, 0.3, 0.5];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_subtract_vectors() {
    let mat = setup();
    let result = mat.calculate("pctr pcvr -", HashMap::new()).unwrap();
    let expected = vec![-0.1, -0.1, -0.1, -0.5];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_multiply_vectors() {
    let mat = setup();
    let result = mat.calculate("pctr pcvr *", HashMap::new()).unwrap();
    let expected = vec![0.02, 0.0, 0.02, 0.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_divide_vectors() {
    let mat = setup();
    let result = mat.calculate("pctr pcvr /", HashMap::new()).unwrap();
    let expected = vec![0.5, 0.0, 0.5, 0.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_divide_vectors_error() {
    let mat = setup();
    let result = mat.calculate("pcvr pctr /", HashMap::new());
    assert!(result.is_err());
}

#[test]
fn test_power_vectors() {
    let mat = setup();
    let result = mat.calculate("pctr pcvr ^", HashMap::new()).unwrap();
    let expected = vec![0.630_957_37, 0.0, 0.630_957_37, 0.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_log_vector() {
    let mat = setup();
    let result = mat.calculate("pcvr log", HashMap::new()).unwrap();
    let expected = vec![-1.609_438, -2.302_585_1, -1.609_438, -0.693_147_2];
    println!("Result: {:?}", result);
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_log_vector_error() {
    let mat = setup();
    let result = mat.calculate("pctr log", HashMap::new());
    assert!(result.is_err());
}

#[test]
fn test_exp_vector() {
    let mat = setup();
    let result = mat.calculate("pcvr exp", HashMap::new()).unwrap();
    let expected = vec![1.221_402_8, 1.105_171, 1.221_402_8, 1.648_721_2];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_abs_vector() {
    let mat = setup();
    let result = mat.calculate("nobygo abs", HashMap::new()).unwrap();
    let expected = vec![0.3, 0.2, 0.5, 0.1];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_percentile_rank_operation() {
    let mat = setup();
    let result = mat
        .calculate("nobygo percentile_rank", HashMap::new())
        .unwrap();
    let expected = vec![0.0, 0.333_333_34, 0.666_666_7, 1.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_or_vectors() {
    let mat = setup();
    let result = mat.calculate("z a |", HashMap::new()).unwrap();
    let expected = vec![0.0, 1.0, 1.0, 1.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_and_vectors() {
    let mat = setup();
    let result = mat.calculate("z a &", HashMap::new()).unwrap();
    let expected = vec![0.0, 0.0, 0.0, 1.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_equal_vectors() {
    let mat = setup();
    let result = mat.calculate("pcvr y ==", HashMap::new()).unwrap();
    let expected = vec![1.0, 0.0, 0.0, 0.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_greater_than_vectors() {
    let mat = setup();
    let result = mat.calculate("pcvr nobygo >", HashMap::new()).unwrap();
    let expected = vec![1.0, 0.0, 1.0, 1.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_greater_than_equal_vectors() {
    let mat = setup();
    let result = mat.calculate("pcvr y >=", HashMap::new()).unwrap();
    let expected = vec![1.0, 0.0, 1.0, 0.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_less_than_vectors() {
    let mat = setup();
    let result = mat.calculate("pcvr nobygo <", HashMap::new()).unwrap();
    let expected = vec![0.0, 1.0, 0.0, 0.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_less_than_equal_vectors() {
    let mat = setup();
    let result = mat.calculate("pcvr y <=", HashMap::new()).unwrap();
    let expected = vec![1.0, 1.0, 0.0, 1.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_min_vectors() {
    let mat = setup();
    let result = mat.calculate("nobygo x min", HashMap::new()).unwrap();
    let expected = vec![-0.3, 0.2, -0.5, 0.1];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_max_vectors() {
    let mat = setup();
    let result = mat.calculate("nobygo x max", HashMap::new()).unwrap();
    let expected = vec![0.6, 0.6, 0.6, 0.6];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_norm_min_max_vector() {
    let mat = setup();
    let result = mat
        .calculate("nobygo norm_min_max", HashMap::new())
        .unwrap();
    println!("Result: {:?}", result);
    let expected = vec![0.285_714_3, 1.0, 0.0, 0.857_142_87];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_norm_percentile_0_99() {
    let mat = setup();
    let result = mat
        .calculate("nobygo norm_percentile_0_99", HashMap::new())
        .unwrap();
    let expected = vec![0.286_944_03, 1.004_304_2, 0.0, 0.860_832_15];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_norm_percentile_5_95() {
    let mat = setup();
    let result = mat
        .calculate("nobygo norm_percentile_5_95", HashMap::new())
        .unwrap();
    let expected = vec![0.259_542, 1.022_900_8, -0.045_801_528, 0.870_229];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_norm_percentile_p_q_error() {
    let mat = setup();
    let result = mat.calculate("nobygo norm_percentile_99_95", HashMap::new());
    assert!(result.is_err());
}
