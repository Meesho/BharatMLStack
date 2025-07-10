use crate::matrix::Mat2D;
use crate::ops::F32Ops;
use crate::vector::Vector;
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
    let expected = vec![0.6309573444801932, 0.0, 0.6309573444801932, 0.0];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_log_vector() {
    let mat = setup();
    let result = mat.calculate("pcvr log", HashMap::new()).unwrap();
    let expected = vec![-1.6094379124341003, -2.302585092994046, -1.6094379124341003, -0.6931471805599453];
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
    let expected = vec![1.2214027581601699, 1.1051709180756477, 1.2214027581601699, 1.6487212707001282];
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
    let result = mat.calculate("nobygo percentile_rank", HashMap::new()).unwrap();
    let expected = vec![0.0, 0.33333333333333, 0.6666666666666, 1.0];
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
    let result = mat.calculate("nobygo norm_min_max", HashMap::new()).unwrap();
    println!("Result: {:?}", result);
    let expected = vec![0.28571428571428575, 1.0, 0.0, 0.8571428571428572];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_norm_percentile_0_99() {
    let mat = setup();
    let result = mat.calculate("nobygo norm_percentile_0_99", HashMap::new()).unwrap();
    let expected = vec![0.2869440459110474, 1.0043041606886658, 0.0, 0.860832137733142];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_norm_percentile_5_95() {
    let mat = setup();
    let result = mat.calculate("nobygo norm_percentile_5_95", HashMap::new()).unwrap();
    let expected = vec![0.2595419847328244, 1.0229007633587786, -0.0458015267175573, 0.8702290076335878];
    assert!(vectors_are_equal(&result, &expected));
}

#[test]
fn test_norm_percentile_p_q_error() {
    let mat = setup();
    let result = mat.calculate("nobygo norm_percentile_99_95", HashMap::new());
    assert!(result.is_err());
}