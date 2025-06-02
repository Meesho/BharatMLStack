job_config = {
    "features_metadata_source_url": "http://host/api/v1/orion/get-source-mapping",  # endpoint to get metadata of features
    "features_output_cloud_storage_path": "gs://orion_features_push_prd/features_push_output",
    "features_write_to_cloud_storage": True,  # Optional. Default: False. If True, the features merged from different offline sources will be written to "features_output_cloud_storage_path"
    
    "kafka_config": {
        "kafka.bootstrap.servers": "localhost:9092",
        "topic": "topic_name",
        "additional_options": {
            "kafka.security.protocol": "SASL_SSL",
            "kafka.sasl.mechanism": "PLAIN",
            "kafka.sasl.jaas.config": "org.apache.kafka.common.security.plain.PlainLoginModule required username='USERNAME' password='PASSWORD';",
            "kafka.compression.type": "zstd",
            "kafka.buffer.memory": "16000000",
            "kafka.batch.size": "61440",
            "kafka.linger.ms": "100",
            "kafka.acks": "1",
            "kafka.retries": "3",
            "kafka.max.in.flight.requests.per.connection": "5",
            "kafka.max.request.size": "1048576",
            "kafka.enable.idempotence": "false",
            "kafka.request.timeout.ms": "30000",
            "kafka.delivery.timeout.ms": "120000",
            "kafka.max.block.ms": "5000",
            "kafka.transaction.timeout.ms": "60000",
            "kafka.retry.backoff.ms": "1000",
            "kafka.metadata.max.age.ms": "300000",
            "max.in.flight.messages": "65536"  # Non-standard
        },
    },
    
    "entities_configs": [  # list of entities to push features
        {  # 1st entity
            "job_id": "user_sscat_features",
            "job_token": "Y2F0YWxvZy1mZW",
            "src_type_to_partition_col_map": {  # Optional. Map of source type to partition column
                "OFS": "process_date",
                "GCS": "ts",
            },
            "fgs_to_consider": [
                "user_sscat_features"
            ],  # Optional. List of feature groups to consider. Default: All feature groups will be considered
            
            "rows_limit": 1000,  # Optional. Rows limit to consider. Default: All rows will be considered
            "push_to_kafka_in_batches": True,  # Optional. Default: False. If True, the features will be pushed to kafka in batches
            "kafka_num_batches": 10  # Optional. Required if push_to_kafka_in_batches is True. Number of batches to push to kafka
        },
        {  # 2nd entity
            "job_id": "catalog_features",
            "job_token": "Y2F0YWxvZy1mZW",
            "src_type_to_partition_col_map": { 
                "OFS": "process_date",
                "GCS": "ts",
            },
            "fgs_to_consider": [
                "user_sscat_features"
            ],  
            "rows_limit": 1000, 
            "push_to_kafka_in_batches": True,  
            "kafka_num_batches": 10
        },
    ],
}
