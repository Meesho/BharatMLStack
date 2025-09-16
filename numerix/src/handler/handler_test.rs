#[cfg(test)]
mod tests {
    use crate::handler::handler::{
        proto, proto::numerix_response_proto, proto::numerix_server::Numerix, ByteList,
        ComputationScoreData, EntityScoreData, MyNumerixService, NumerixRequestProto,
        NumerixResponseProto, Score, StringList,
    };
    use tonic::Request;

    use crate::handler::config as handler_config;
    use crate::pkg::config::config;
    use crate::pkg::etcd::etcd;
    use crate::pkg::logger::logger;
    use crate::pkg::metrics::metrics;
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::LazyLock;
    use tokio::sync::Mutex;

    // For logger and sync initialization
    static LOGGER_INITIALIZED: std::sync::Once = std::sync::Once::new();

    // For etcd client initialization protection
    static ETCD_MUTEX: LazyLock<Mutex<()>> = LazyLock::new(|| Mutex::new(()));
    static ETCD_INITIALIZED: AtomicBool = AtomicBool::new(false);

    fn create_test_request(
        schema: Vec<String>,
        scores: Vec<Score>,
        compute_id: String,
        data_type: Option<String>,
    ) -> NumerixRequestProto {
        NumerixRequestProto {
            entity_score_data: Some(EntityScoreData {
                schema,
                entity_scores: scores,
                compute_id,
                data_type,
            }),
        }
    }

    async fn setup() {
        // Initialize sync components once
        LOGGER_INITIALIZED.call_once(|| {
            logger::init_logger();
            config::get_config();
            metrics::init_config();
        });

        // Initialize etcd client only once with proper locking
        if !ETCD_INITIALIZED.load(Ordering::SeqCst) {
            // Acquire the mutex to ensure only one thread initializes etcd
            let _guard = ETCD_MUTEX.lock().await;

            // Check again after acquiring the lock
            if !ETCD_INITIALIZED.load(Ordering::SeqCst) {
                println!("Initializing etcd client...");
                etcd::init_etcd_connection().await;
                handler_config::init_config().await;
                ETCD_INITIALIZED.store(true, Ordering::SeqCst);
                println!("Etcd client initialized successfully");
            }
        }
    }

    #[tokio::test]
    async fn test_valid_compute_request() {
        setup().await;
        let service = MyNumerixService::default();

        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["1.0".to_string(), "2.0".to_string()],
                    })),
                },
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["3.0".to_string(), "4.0".to_string()],
                    })),
                },
            ],
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();
        let response_check = NumerixResponseProto {
            response: Some(numerix_response_proto::Response::ComputationScoreData(
                ComputationScoreData {
                    schema: vec!["a".to_string(), "score".to_string()], // The original first column plus "score" column
                    computation_scores: vec![
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["3".to_string()], // The computed result as a string
                                },
                            )),
                        },
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["7".to_string()], // The computed result as a string
                                },
                            )),
                        },
                    ],
                },
            )),
        };
        println!("response_check: {:?}", response_check);
        println!("response_inner: {:?}", response_inner);
        assert!(
            response_check == response_inner,
            "Expected computation score data in response"
        );
    }

    #[tokio::test]
    async fn test_valid_compute_request_with_byte_data() {
        setup().await;
        let service = MyNumerixService::default();
        let num1: f64 = 1.0;
        let num2: f64 = 2.0;
        let num3: f64 = 3.0;
        let num4: f64 = 4.0;
        let num5: f64 = 3.0;
        let num6: f64 = 7.0;
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::ByteData(ByteList {
                        values: vec![num2.to_le_bytes().to_vec(), num1.to_le_bytes().to_vec()], // Representing 1.0 and 2.0 as bytes
                    })),
                },
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::ByteData(ByteList {
                        values: vec![num4.to_le_bytes().to_vec(), num3.to_le_bytes().to_vec()], // Representing 3.0 and 4.0 as bytes
                    })),
                },
            ],
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();
        let response_check = NumerixResponseProto {
            response: Some(numerix_response_proto::Response::ComputationScoreData(
                ComputationScoreData {
                    schema: vec!["a".to_string(), "score".to_string()],
                    computation_scores: vec![
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::ByteData(ByteList {
                                values: vec![num5.to_le_bytes().to_vec()], // Representing 3.0 as bytes
                            })),
                        },
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::ByteData(ByteList {
                                values: vec![num6.to_le_bytes().to_vec()], // Representing 7.0 as bytes
                            })),
                        },
                    ],
                },
            )),
        };
        println!("response_check: {:?}", response_check);
        println!("response_inner: {:?}", response_inner);
        assert!(
            response_check == response_inner,
            "Expected computation score data in response with byte data"
        );
    }

    #[tokio::test]
    async fn test_f32_data_type() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with f32 data type
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["1.5".to_string(), "2.5".to_string()],
                    })),
                },
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["3.5".to_string(), "4.5".to_string()],
                    })),
                },
            ],
            "1".to_string(),
            Some("f32".to_string()), // Explicitly specify f32 data type
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expected response with calculation done as f32
        let response_check = NumerixResponseProto {
            response: Some(numerix_response_proto::Response::ComputationScoreData(
                ComputationScoreData {
                    schema: vec!["a".to_string(), "score".to_string()],
                    computation_scores: vec![
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["4".to_string()],
                                },
                            )),
                        },
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["8".to_string()],
                                },
                            )),
                        },
                    ],
                },
            )),
        };

        println!("f32 response_check: {:?}", response_check);
        println!("f32 response_inner: {:?}", response_inner);
        assert!(
            response_check == response_inner,
            "Expected computation score data in response with f32 data type"
        );
    }

    #[tokio::test]
    async fn test_invalid_compute_id() {
        setup().await;
        let service = MyNumerixService::default();

        // Create request with invalid compute_id
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec!["1.0".to_string(), "2.0".to_string()],
                })),
            }],
            "invalid_compute_id".to_string(), // This ID doesn't exist in config
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expect an error response for invalid compute_id
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(error)) => {
                assert_eq!(error.message, "Expression not found");
            }
            _ => panic!(
                "Expected error response for invalid compute_id, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_missing_entity_score_data() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with missing entity_score_data
        let request = NumerixRequestProto {
            entity_score_data: None,
        };

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expect an error response for missing entity_score_data
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(error)) => {
                assert_eq!(error.message, "Missing entity_score_data");
            }
            _ => panic!(
                "Expected error response for missing entity_score_data, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_empty_schema() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with empty schema
        let request = create_test_request(
            vec![], // Empty schema
            vec![Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec!["1.0".to_string(), "2.0".to_string()],
                })),
            }],
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expect an error response for empty schema
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(error)) => {
                assert_eq!(error.message, "Empty schema");
            }
            _ => panic!(
                "Expected error response for empty schema, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_empty_entity_scores() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with empty entity_scores
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![], // Empty entity scores
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expect an error response for empty entity_scores
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(error)) => {
                assert_eq!(error.message, "Empty entity_scores");
            }
            _ => panic!(
                "Expected error response for empty entity_scores, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_empty_compute_id() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with empty compute_id
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec!["1.0".to_string(), "2.0".to_string()],
                })),
            }],
            "".to_string(), // Empty compute_id
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expect an error response for empty compute_id
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(error)) => {
                assert_eq!(error.message, "Empty compute_id");
            }
            _ => panic!(
                "Expected error response for empty compute_id, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_invalid_string_data_format() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with invalid string data (not parsable as numbers)
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec!["not_a_number".to_string(), "2.0".to_string()],
                })),
            }],
            "1".to_string(),
            Some("f64".to_string()),
        );

        // Service should handle the error gracefully by using default values (0)
        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // The computation should still proceed with default values for unparseable inputs
        match response_inner.response {
            Some(numerix_response_proto::Response::ComputationScoreData(_)) => {
                // Test passes if we got computation data instead of an error
            }
            _ => panic!(
                "Expected computation data with default values, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_larger_dataset() {
        setup().await;
        let service = MyNumerixService::default();

        // Create 10 scores to test with a larger dataset
        let mut scores = Vec::new();
        for i in 0..10 {
            scores.push(Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec![format!("{}.0", i), format!("{}.0", i + 1)],
                })),
            });
        }

        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            scores,
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Check if we have the expected number of results (10)
        match &response_inner.response {
            Some(numerix_response_proto::Response::ComputationScoreData(data)) => {
                assert_eq!(
                    data.computation_scores.len(),
                    10,
                    "Expected 10 computation results"
                );
            }
            _ => panic!("Expected computation data, got: {:?}", response_inner),
        }
    }

    #[tokio::test]
    async fn test_no_data_type_specified() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request without specifying data_type (should default to f64)
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["1.0".to_string(), "2.0".to_string()],
                    })),
                },
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["3.0".to_string(), "4.0".to_string()],
                    })),
                },
            ],
            "1".to_string(),
            None, // No data_type specified, should default to f64
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expected response should be the same as f64 calculation
        let response_check = NumerixResponseProto {
            response: Some(numerix_response_proto::Response::ComputationScoreData(
                ComputationScoreData {
                    schema: vec!["a".to_string(), "score".to_string()],
                    computation_scores: vec![
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["3".to_string()],
                                },
                            )),
                        },
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["7".to_string()],
                                },
                            )),
                        },
                    ],
                },
            )),
        };

        assert!(
            response_check == response_inner,
            "Expected computation score data with default f64 data type"
        );
    }

    #[tokio::test]
    async fn test_negative_values() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with negative values
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["-1.0".to_string(), "-2.0".to_string()],
                    })),
                },
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["-3.0".to_string(), "-4.0".to_string()],
                    })),
                },
            ],
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expected response with negative values
        let response_check = NumerixResponseProto {
            response: Some(numerix_response_proto::Response::ComputationScoreData(
                ComputationScoreData {
                    schema: vec!["a".to_string(), "score".to_string()],
                    computation_scores: vec![
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["-3".to_string()],
                                },
                            )),
                        },
                        Score {
                            matrix_format: Some(proto::score::MatrixFormat::StringData(
                                StringList {
                                    values: vec!["-7".to_string()],
                                },
                            )),
                        },
                    ],
                },
            )),
        };

        assert!(
            response_check == response_inner,
            "Expected computation score data with negative values"
        );
    }

    #[tokio::test]
    async fn test_matrix_format_none() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request with a score that has no matrix_format specified
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![Score {
                matrix_format: None, // No matrix format specified
            }],
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // Expect response to contain an error or handle the missing format gracefully
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(_)) => {
                // Test passes if we got an error response
            }
            Some(numerix_response_proto::Response::ComputationScoreData(_)) => {
                // Or test passes if service handled it by using default values
            }
            _ => panic!(
                "Expected either error or computation data with defaults, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_mismatched_schema_and_values() {
        setup().await;
        let service = MyNumerixService::default();

        // Create a request where schema length doesn't match values length
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string(), "c".to_string()], // 3 columns in schema
            vec![Score {
                matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                    values: vec!["1.0".to_string(), "2.0".to_string()], // But only 2 values provided
                })),
            }],
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // The service should either return an error or handle the mismatch gracefully
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(_)) => {
                // Test passes if we got an error response
            }
            Some(numerix_response_proto::Response::ComputationScoreData(_)) => {
                // Or test passes if service handled it by using default values for missing data
            }
            _ => panic!(
                "Expected either error or computation data with defaults, got: {:?}",
                response_inner
            ),
        }
    }

    #[tokio::test]
    async fn test_mixed_data_types() {
        setup().await;
        let service = MyNumerixService::default();

        // Get byte representation of some f64 values
        let num1: f64 = 1.0;
        let num2: f64 = 2.0;

        // Create a request with mixed string and byte data types
        let request = create_test_request(
            vec!["a".to_string(), "b".to_string()],
            vec![
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::StringData(StringList {
                        values: vec!["1.0".to_string(), "2.0".to_string()],
                    })),
                },
                Score {
                    matrix_format: Some(proto::score::MatrixFormat::ByteData(ByteList {
                        values: vec![num1.to_le_bytes().to_vec(), num2.to_le_bytes().to_vec()],
                    })),
                },
            ],
            "1".to_string(),
            Some("f64".to_string()),
        );

        let response = service.compute(Request::new(request)).await.unwrap();
        let response_inner = response.into_inner();

        // The service should either return an error or handle the mixed types gracefully
        // For example, converting all to a consistent format for calculation
        match response_inner.response {
            Some(numerix_response_proto::Response::Error(_)) => {
                // Test passes if we got an error response
            }
            Some(numerix_response_proto::Response::ComputationScoreData(_)) => {
                // Or test passes if service handled the mixed types
            }
            _ => panic!(
                "Expected either error or computation data, got: {:?}",
                response_inner
            ),
        }
    }
}
