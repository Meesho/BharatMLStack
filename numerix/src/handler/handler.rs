use tonic::{Request, Response, Status};
use std::time::Instant;
use crate::pkg::metrics::metrics;
use crate::handler::config;
use std::collections::HashMap;
use crate::pkg::rust_matrix_frame::matrix::Mat2D;
use crate::pkg::rust_matrix_frame::ops::F32Ops;
use crate::pkg::rust_matrix_frame::ops::F64Ops;
use crate::pkg::rust_matrix_frame::vector::Vector;
use crate::pkg::rust_matrix_frame::ops::VectorOps;
use crate::pkg::rust_matrix_frame::error::Mat2DError;
use bytemuck::Pod;
use std::str::FromStr;
use crate::logger;
use ryu;
pub mod proto {
    include!("../protos/proto_gen/numerix.rs");
}


pub use proto::{
    NumerixRequestProto,
    NumerixResponseProto,
    ComputationScoreData,
    numerix_response_proto,
    Error,
    Score,
    StringList,
    ByteList,
    EntityScoreData,
};

pub use proto::numerix_server::Numerix;

static COMPUTE_ID: &str = "compute_id";
static DEFAULT_DATA_TYPE: &str = "fp64";
static STRING_SCORE_TYPE: &str = "String";
static BYTE_SCORE_TYPE: &str = "Byte";


#[derive(Debug, Default)]
pub struct MyNumerixService;

#[tonic::async_trait]
impl Numerix for MyNumerixService {
    async fn compute(
        &self,
        request: Request<NumerixRequestProto>,
    ) -> Result<Response<NumerixResponseProto>, Status> {
        let start = Instant::now();
        let req = request.into_inner();
        match validate_request(&req) {
            Ok(_) => (),
            Err(e) => {
                logger::error(format!("Invalid request: {:?}, Error: {}", req, e), None);
                return Ok(Response::new(NumerixResponseProto {
                    response: Some(numerix_response_proto::Response::Error(Error {
                        message: format!("Request validation failed: {}", e),
                    })),
                }));
            }
        }

        let entity_score_data = req.entity_score_data.as_ref().unwrap();
        let compute_id = entity_score_data.compute_id.clone();
        let tags = vec![(COMPUTE_ID, compute_id.as_str())];

        let _ = metrics::count("numerix.computation.request.total", 1, &tags);

        let expression = config::get_exression(compute_id.as_str());
        if expression.is_empty() {
            return Ok(Response::new(NumerixResponseProto {
                response: Some(numerix_response_proto::Response::Error(Error {
                    message: format!("No expression configured for compute_id: {}", compute_id),
                })),
            }));
        }
        
        let result = compute_expression(&expression, &req);
        if let Some(numerix_response_proto::Response::Error(_)) = &result.response {
            let _ = metrics::count("numerix.computation.request.error", 1, &tags);
            return Ok(Response::new(result));
        }

        let duration = start.elapsed();
        let _ = metrics::timing("numerix.computation.request.latency", duration, &tags);

        Ok(Response::new(result))
    }
}

fn validate_request(req: &NumerixRequestProto) -> Result<(), String> {
    if req.entity_score_data.is_none() {
        return Err("Missing entity_score_data".to_string());
    }

    let entity_score_data = req.entity_score_data.as_ref().unwrap();

    if entity_score_data.compute_id.is_empty() {
        return Err("Missing compute_id".to_string());
    }

    if entity_score_data.entity_scores.is_empty() {
        return Err("Missing entity_scores".to_string());
    }

    if entity_score_data.schema.is_empty() {
        return Err("Missing schema".to_string());
    }

    if entity_score_data.entity_scores[0].matrix_format.is_none() {
        return Err("Missing data in entity_scores".to_string());
    }

    Ok(())
}

fn compute_expression(expression: &str, req: &NumerixRequestProto) -> NumerixResponseProto {
    let entity_score_data = req.entity_score_data.as_ref().unwrap();
    let compute_id = entity_score_data.compute_id.clone();
    let conversion_type = entity_score_data.data_type.clone().unwrap_or_else(|| DEFAULT_DATA_TYPE.to_string());

    let score_type = if let Some(first_score) = entity_score_data.entity_scores.first() {
        match &first_score.matrix_format {
            Some(proto::score::MatrixFormat::StringData(_)) => STRING_SCORE_TYPE,
            Some(proto::score::MatrixFormat::ByteData(_)) => BYTE_SCORE_TYPE,
            None => STRING_SCORE_TYPE,
        }
    } else {
        STRING_SCORE_TYPE
    };

    let column_names: HashMap<String, usize> = entity_score_data.schema.clone()
        .iter()
        .enumerate()
        .map(|(i, s)| (s.clone(), i))
        .collect();


    if conversion_type == DEFAULT_DATA_TYPE {
        compute_scores::<f64, F64Ops>(column_names, expression, score_type, req, compute_id.as_str())
    }
    else {
        compute_scores::<f32, F32Ops>(column_names, expression, score_type, req, compute_id.as_str())
    }
}



fn compute_scores<T, Ops>(
    column_names: HashMap<String, usize>,
    expression: &str,
    score_type: &str,
    req: &NumerixRequestProto,
    compute_id: &str,
) -> NumerixResponseProto
where
    T: Copy + FromStr + Default + From<u8> + Pod + std::fmt::Display + PartialOrd + std::fmt::Debug + FromBytes + FastToString,
    <T as FromStr>::Err: std::fmt::Debug,
    Vector<T>: Clone,
    Ops: VectorOps<Scalar = T>,
    Ops::Scalar: Default + Copy + FromStr + std::fmt::Display,
    <Ops::Scalar as FromStr>::Err: std::fmt::Debug,
{

    let cols = column_names.len();
    let rows = req.entity_score_data.as_ref().unwrap().entity_scores.len();
    let converted_scores = convert_scores::<T>(score_type, req, rows, cols);
    let meta_data = meta_data_from_compute_id::<T>(compute_id, rows);
    let matrix = Mat2D::<Ops>::from_data(rows, cols, converted_scores, column_names);
    let matrix = match matrix {
        Ok(matrix) => matrix,
        Err(e) => {
            logger::error(format!("Failed to create matrix for request: {:?}", req), Some(&e));
            return NumerixResponseProto {
                response: Some(numerix_response_proto::Response::Error(Error {
                    message: format!("Matrix setup failed for request: {:?}", req),
                })),
            };
        }
    };
    let result = matrix.calculate(expression, meta_data);

    convert_to_grpc_response::<T>(result, req, score_type)
}

trait FromBytes: Sized {
    fn from_le_bytes(bytes: &[u8]) -> Self;
}

impl FromBytes for f32 {
    fn from_le_bytes(bytes: &[u8]) -> Self {
        f32::from_le_bytes(bytes.try_into().expect("Invalid byte length"))
    }
}

impl FromBytes for f64 {
    fn from_le_bytes(bytes: &[u8]) -> Self {
        f64::from_le_bytes(bytes.try_into().expect("Invalid byte length"))
    }
}

fn convert_scores<T>(score_type: &str, req: &NumerixRequestProto, rows: usize, cols: usize) -> Vec<T>
where
    T: std::str::FromStr + Copy + Default + From<u8> + FromBytes + std::fmt::Debug,
    <T as std::str::FromStr>::Err: std::fmt::Debug,
{
    let mut converted_scores = Vec::with_capacity(rows * cols);
    converted_scores.resize(rows * cols, T::default());

    let entity_scores = &req.entity_score_data.as_ref().unwrap().entity_scores;

    match score_type {
        "Byte" => {
            for (idx, score_list) in entity_scores.iter().enumerate() {
                let byte_data = match &score_list.matrix_format {
                    Some(proto::score::MatrixFormat::ByteData(data)) => data,
                    _ => continue,
                };
                
                for (value_idx, value) in byte_data.values.iter().enumerate().skip(1) {
                    if value_idx < cols && idx < rows {
                        let expected_size = std::mem::size_of::<T>();
                        if value.len() != expected_size {
                            logger::error(format!("Invalid byte length: expected {} bytes, got {} bytes for request: {:?}", 
                                expected_size, value.len(), req), None);
                            continue;
                        }
                        converted_scores[value_idx * rows + idx] = T::from_le_bytes(value);
                    }
                }
            }
        }
        "String" => {
            for (idx, score_list) in entity_scores.iter().enumerate() {
                let string_data = match &score_list.matrix_format {
                    Some(proto::score::MatrixFormat::StringData(data)) => data,
                    _ => continue,
                };
                
                for (value_idx, value) in string_data.values.iter().enumerate().skip(1) {
                    if value_idx < cols && idx < rows {
                        match value.parse::<T>() {
                            Ok(parsed_value) => converted_scores[value_idx * rows + idx] = parsed_value,
                            Err(e) => {
                                logger::error(format!("Failed to parse string value '{}' for request: {:?}, parse error: {:?}", value, req, e), None);
                            }
                        }
                    }
                }
            }
        }
        _ => {}
    }

    converted_scores
}

fn meta_data_from_compute_id<T>(compute_id: &str, size: usize) -> HashMap<String, Vector<T>>
where
    T: FromStr + Copy + Default,
    <T as FromStr>::Err: std::fmt::Debug,
{
    let meta_data = config::get_meta_data(compute_id);
    let mut numbers_map = HashMap::new();

    for number_str in meta_data {
        match number_str.parse::<T>() {
            Ok(number_value) => {
                numbers_map.entry(number_str).or_insert(Vector::from_vec(vec![number_value; size]));
            },
            Err(e) => {
                logger::error(
                    format!("Failed to parse number '{}' in expression for compute_id: {}: {:?}", number_str, compute_id, e),
                    None
                );
                continue;
            }
        }
    }

    numbers_map
}

trait FastToString {
    fn fast_to_string(&self) -> String;
}

impl FastToString for f32 {
    fn fast_to_string(&self) -> String {
        ryu::Buffer::new().format(*self).to_string()
    }
}

impl FastToString for f64 {
    fn fast_to_string(&self) -> String {
        ryu::Buffer::new().format(*self).to_string()
    }
}

fn convert_to_grpc_response<T>(
    result: Result<Vector<T>, Mat2DError>,
    req: &NumerixRequestProto,
    score_type: &str,
) -> NumerixResponseProto
where
    T: ToString + Copy + Pod + FastToString,
{
    let entity_score_data = req.entity_score_data.as_ref().unwrap();

    let result_vec = match result {
        Ok(result_vec) => result_vec,
        Err(err) => {
            logger::error(format!("Matrix calculation failed for request: {:?}", req), Some(&err));
            return NumerixResponseProto {
                response: Some(numerix_response_proto::Response::Error(Error {
                    message: format!("Calculation failed for request '{:?}' : {}", req, err),
                })),
            };
        }
    };

    let schema = vec![
        entity_score_data.schema[0].clone(),
        "score".into(),
    ];

    let result_slice = result_vec.as_slice();
    let mut computation_scores = Vec::with_capacity(result_slice.len());

    let score_type_byte = score_type.as_bytes();
    if score_type_byte == b"String" {
        for (i, value) in result_slice.iter().enumerate() {
            let catalog_id = match &entity_score_data.entity_scores[i].matrix_format {
                Some(proto::score::MatrixFormat::StringData(data)) => {
                    if !data.values.is_empty() {
                        data.values[0].clone()
                    } else {
                        String::new()
                    }
                },
                _ => String::new(),
            };
            
            let value_string = value.fast_to_string();
            
            computation_scores.push(Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec![catalog_id, value_string],
                })),
            });
        }
    } else if score_type_byte == b"Byte" {
        for (i, value) in result_slice.iter().enumerate() {
            let catalog_id = match &entity_score_data.entity_scores[i].matrix_format {
                Some(proto::score::MatrixFormat::ByteData(data)) => {
                    if !data.values.is_empty() {
                        data.values[0].clone()
                    } else {
                        Vec::new()
                    }
                },
                _ => Vec::new(),
            };
            
            let bytes: &[u8] = bytemuck::bytes_of(value);
            computation_scores.push(Score {
                matrix_format: Some(proto::score::MatrixFormat::ByteData(ByteList {
                    values: vec![catalog_id, bytes.to_vec()],
                })),
            });
        }
    } else {
        for (i, value) in result_slice.iter().enumerate() {
            let catalog_id = match &entity_score_data.entity_scores[i].matrix_format {
                Some(proto::score::MatrixFormat::StringData(data)) => {
                    if !data.values.is_empty() {
                        data.values[0].clone()
                    } else {
                        String::new()
                    }
                },
                _ => String::new(),
            };
            
            let value_string = value.fast_to_string();
            
            computation_scores.push(Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec![catalog_id, value_string],
                })),
            });
        }
    }

    let entity_score_data = ComputationScoreData {
        schema,
        computation_scores,
    };

    NumerixResponseProto {
        response: Some(numerix_response_proto::Response::ComputationScoreData(entity_score_data)),
    }
}




