from .utils.helpers import get_features_details
import numpy as np


class OnlineFeatureStorePyClient:

    def __init__(self, features_metadata_source_url: str, job_id: str, job_token: str, fgs_to_consider: list = []):
        self.features_metadata_source_url = features_metadata_source_url
        self.job_id = job_id
        self.job_token = job_token
        self.fgs_to_consider = fgs_to_consider
        
        (
            offline_src_type_columns,
            offline_col_to_default_values_map,
            onfs_fg_to_onfs_feat_map,
            onfs_fg_to_ofs_feat_map,
            fg_to_datatype_map,
            entity_label,
            entity_column_names,
            offline_col_to_datatype_map,
        ) = get_features_details(
            self.features_metadata_source_url,
            self.job_id,
            self.job_token,
            self.fgs_to_consider,
        )
        self.offline_src_type_columns = offline_src_type_columns
        self.offline_col_to_default_values_map = offline_col_to_default_values_map
        self.onfs_fg_to_onfs_feat_map = onfs_fg_to_onfs_feat_map
        self.onfs_fg_to_ofs_feat_map = onfs_fg_to_ofs_feat_map
        self.fg_to_datatype_map = fg_to_datatype_map
        self.entity_label = entity_label
        self.entity_column_names = entity_column_names
        self.offline_col_to_datatype_map = offline_col_to_datatype_map
        
    def get_offline_col_to_datatype_map(self):
        return self.offline_col_to_datatype_map

    def get_features_details(self):
        return (
            self.offline_src_type_columns,
            self.offline_col_to_default_values_map,
            self.entity_column_names,
        )

    def generate_df_with_protobuf_messages(self, df, intra_batch_size: int = 20):
        

        from pyspark.sql.types import StructType, StructField, BinaryType, LongType

        def process_partition(iterator):
            """Convert each partition of Spark DataFrame into Protobuf serialized messages."""

            # Import protobuf files from bharatml_commons
            from bharatml_commons.proto.persist.persist_pb2 import (
                Query,
                FeatureGroupSchema,
                Data,
                FeatureValues,
                Values,
                Vector,
            )

            feature_group_schema = [
                FeatureGroupSchema(label=label, feature_labels=features)
                for label, features in self.onfs_fg_to_onfs_feat_map.items()
            ]

            current_batch = []
            batch_id = 0

            for row in iterator:
                feature_values = []
                for fg_label, features in self.onfs_fg_to_ofs_feat_map.items():
                    curr_datatype = self.fg_to_datatype_map[fg_label]

                    values = Values()
                    # For Scalar Data types
                    
                    if curr_datatype == "DataTypeFP8E5M2":
                        values.fp32_values.extend(
                            [np.float32(row[feature]) for feature in features]
                        )
                        
                    if curr_datatype == "DataTypeFP16":
                        values.fp32_values.extend(
                            [np.float32(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeFP32":
                        values.fp32_values.extend(
                            [np.float32(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeFP64":
                        values.fp64_values.extend(
                            [np.float64(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeInt32":
                        values.int32_values.extend(
                            [np.int32(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeInt64":
                        values.int64_values.extend(
                            [np.int64(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeUInt32":
                        values.uint32_values.extend(
                            [np.uint32(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeUInt64":
                        values.uint64_values.extend(
                            [np.uint64(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeString":
                        values.string_values.extend(
                            [str(row[feature]) for feature in features]
                        )
                    elif curr_datatype == "DataTypeBool":
                        values.bool_values.extend(
                            [bool(row[feature]) for feature in features]
                        )

                    # For Vector Data types
                    elif curr_datatype == "DataTypeFP16Vector":
                        for feature in features:
                            vector_values = Values(
                                fp32_values=[np.float32(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeFP8E5M2Vector":
                        for feature in features:
                            vector_values = Values(
                                fp32_values=[np.float32(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeFP8E4M3Vector":
                        for feature in features:
                            vector_values = Values(
                                fp32_values=[np.float32(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))                            
                    elif curr_datatype == "DataTypeFP32Vector":
                        for feature in features:
                            vector_values = Values(
                                fp32_values=[np.float32(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeFP64Vector":
                        for feature in features:
                            vector_values = Values(
                                fp64_values=[np.float64(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeInt32Vector":
                        for feature in features:
                            vector_values = Values(
                                int32_values=[np.int32(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeInt64Vector":
                        for feature in features:
                            vector_values = Values(
                                int64_values=[np.int64(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeUInt32Vector":
                        for feature in features:
                            vector_values = Values(
                                uint32_values=[np.uint32(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeUInt64Vector":
                        for feature in features:
                            vector_values = Values(
                                uint64_values=[np.uint64(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeStringVector":
                        for feature in features:
                            vector_values = Values(
                                string_values=[str(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))
                    elif curr_datatype == "DataTypeBoolVector":
                        for feature in features:
                            vector_values = Values(
                                bool_values=[bool(x) for x in row[feature]]
                            )
                            values.vector.append(Vector(values=vector_values))

                    feature_values.append(FeatureValues(values=values))

                # Construct Data message for current row
                data_msg = Data(
                    key_values=[str(row[col]) for col in self.entity_column_names],
                    feature_values=feature_values,
                )

                current_batch.append(data_msg)

                # When batch is full or at end of iterator, create and yield Query message
                if len(current_batch) >= intra_batch_size:
                    query = Query(
                        entity_label=self.entity_label,
                        keys_schema=self.entity_column_names,
                        feature_group_schema=feature_group_schema,
                        data=current_batch,
                    )
                    yield (query.SerializeToString(), batch_id)
                    current_batch = []
                    batch_id += 1

            # Handle any remaining items in the last batch
            if current_batch:
                query = Query(
                    entity_label=self.entity_label,
                    keys_schema=self.entity_column_names,
                    feature_group_schema=feature_group_schema,
                    data=current_batch,
                )
                yield (query.SerializeToString(), batch_id)

        # Define output schema
        protobuf_schema = StructType([
            StructField("value", BinaryType(), False),
            StructField("intra_batch_id", LongType(), False)
        ])

        # Apply mapPartitions
        out_df = df.rdd.mapPartitions(process_partition).toDF(protobuf_schema)
        return out_df

    def write_protobuf_df_to_kafka(
        self,
        df,
        kafka_bootstrap_servers: str,
        kafka_topic: str,
        additional_options: dict = {},
        kafka_num_batches: int = 1,
    ):
        additional_options = additional_options or {}

        # Base Kafka config
        kafka_config = {
            "kafka.bootstrap.servers": kafka_bootstrap_servers,
            "topic": kafka_topic,
        }

        # Merge base config with provided options
        kafka_config.update(additional_options)

        
        # Write to Kafka
        if kafka_num_batches == 1:
            df = df.drop("intra_batch_id")
            df.write.format("kafka").options(**kafka_config).save()
        else:
            import pyspark.sql.functions as F
            print("Writing to Kafka in batches")
            df = df.withColumn("batch_no", F.col("intra_batch_id") % kafka_num_batches)
            df = df.drop("intra_batch_id")
            
            for i in range(kafka_num_batches):
                df_batch = df.filter(F.col("batch_no") == i)
                df_batch.write.format("kafka").options(**kafka_config).save()
                print(f"Wrote batch {i} to Kafka")
