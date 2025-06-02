from helpers import get_latest_path
from functools import reduce
from pyspark.sql import functions as F

def write_to_cloud_storage(df, out_path):
    df.write.format("parquet").mode("overwrite").save(out_path)


def read_from_source(spark, source, source_type, features, partition_col):
    """
    Reads data from a single source into a Spark DataFrame.
    Args:
        spark (SparkSession): The active Spark session
        source (str): Source location - can be a table name or cloud storage path
        source_type (str): Type of source. One of:
            - "TABLE": Hive/Delta table
            - "PARQUET_GCS": Parquet files in Google Cloud Storage
            - "PARQUET_S3": Parquet files in AWS S3
            - "PARQUET_ADLS": Parquet files in Azure Data Lake Storage
            - "DELTA_GCS": Delta files in Google Cloud Storage
            - "DELTA_S3": Delta files in AWS S3
            - "DELTA_ADLS": Delta files in Azure Data Lake Storage
        features (list): List of feature column names to select
        partition_col (str): Name of partition column to filter on

    Returns:
        pyspark.sql.DataFrame: DataFrame containing the selected features from the source,
        filtered to the latest partition and deduplicated.

    Raises:
        ValueError: If source_type is not one of the supported types
    """

    if source_type == "TABLE":
        max_partition_col = str(
            spark.sql(f"select max({partition_col}) from {source}").collect()[0][0]
        )
        print(f"reading from table: {source} with max partition_col: {max_partition_col}")
        sdf = (
            spark.sql(
                f"select * from {source} where {partition_col} = '{max_partition_col}'"
            )
            .select(*features)
            .distinct()
        )

    elif source_type in ["PARQUET_GCS", "PARQUET_S3", "PARQUET_ADLS"]:
        latest_path = get_latest_path(source, partition_col, "parquet")
        sdf = spark.read.parquet(latest_path).select(*features).distinct()

    elif source_type in ["DELTA_GCS", "DELTA_S3", "DELTA_ADLS"]:
        latest_path = get_latest_path(source, partition_col, "delta")
        sdf = spark.read.format("delta").load(latest_path).select(*features).distinct()

    else:
        raise ValueError(f"Unsupported source_type: {source_type}")

    return sdf


def get_features_from_all_sources(
    spark,
    entity_column_names,
    feature_mapping,
    src_type_to_partition_col_map={
        "TABLE": "process_date",
        "PARQUET_GCS": "ts",
        "PARQUET_S3": "ts",
        "PARQUET_ADLS": "ts",
        "DELTA_GCS": "ts",
        "DELTA_S3": "ts",
        "DELTA_S3": "ts",
    },
):
    """
    Retrieves features from multiple data sources and combines them into a single DataFrame.

    This function reads features from different sources (tables, parquet files, delta files),
    performs necessary transformations like renaming columns, joins the data from different
    sources on entity columns, and handles null values.

    Args:
        spark: Spark session object
        entity_column_names (list): List of column names that uniquely identify an entity
        feature_mapping (list): List of tuples containing source details:
            [(source_path, source_type, [(source_col, renamed_col), ...]), ...]
        src_type_to_partition_col_map (dict): Mapping of source types to their partition columns.
            Defaults to:
            {
                "TABLE": "process_date",
                "PARQUET_GCS": "ts",
                "PARQUET_S3": "ts",
                "PARQUET_ADLS": "ts",
                "DELTA_GCS": "ts",
                "DELTA_S3": "ts",
                "DELTA_S3": "ts"
            }

    Returns:
        pyspark.sql.DataFrame: DataFrame containing all features from different sources,
            joined on entity columns, with null values handled.

    Example:
        >>> feature_mapping = [
        ...     ("my_table", "TABLE", [("feat1", "renamed_feat1")]),
        ...     ("gs://bucket/path", "PARQUET_GCS", [("feat2", "renamed_feat2")])
        ... ]
        >>> df = get_features_from_all_sources(
        ...     spark,
        ...     ["entity_id"],
        ...     feature_mapping,
        ...     {"renamed_feat1": 0, "renamed_feat2": ""}
        ... )
    """

    # List to store DataFrames
    sdf_list = []

    # Loop over all feature mappings, including index 0
    for i in range(len(feature_mapping)):
        source = feature_mapping[i][0]
        source_type = feature_mapping[i][1]
        src_features = [x[0] for x in feature_mapping[i][2]]
        renamed_features = [x[1] for x in feature_mapping[i][2]]

        features = entity_column_names + src_features

        print(f"Source {i+1}: {source}")

        # Read the current table
        sdf_new = read_from_source(
            spark,
            source,
            source_type,
            features,
            src_type_to_partition_col_map[source_type],
        )

        # Rename columns
        rename_mapping = {
            src: dest for src, dest in zip(src_features, renamed_features)
        }
        sdf_new = sdf_new.select(
            [F.col(col).alias(rename_mapping.get(col, col)) for col in sdf_new.columns]
        )

        # Add the DataFrame to the list
        sdf_list.append(sdf_new)

    if len(sdf_list) == 1:
        sdf = sdf_list[0]
    else:
        # Perform a single multi-way join using reduce
        print("Joining all feature sources...")
        sdf = reduce(
            lambda df1, df2: df1.join(df2, entity_column_names, "outer"), sdf_list
        )

    print("filling null features with default values")
    
    sdf = sdf.na.drop(subset=entity_column_names)

    return sdf

def fill_na_features_with_default_values(df, offline_col_to_default_values_map, offline_col_to_datatype_map):
    # Verify maps are non-empty and have matching keys
    assert offline_col_to_datatype_map, "offline_col_to_datatype_map is empty"
    assert offline_col_to_default_values_map, "offline_col_to_default_values_map is empty"
    assert set(offline_col_to_datatype_map.keys()) == set(offline_col_to_default_values_map.keys()), \
        "Keys in offline_col_to_datatype_map and offline_col_to_default_values_map do not match"
    
    vector_types = {
        'DataTypeFP8E5M2Vector', 'DataTypeFP8E4M3Vector', 'DataTypeFP16Vector', 
        'DataTypeFP32Vector', 'DataTypeFP64Vector', 'DataTypeInt8Vector', 
        'DataTypeInt16Vector', 'DataTypeInt32Vector', 'DataTypeInt64Vector',
        'DataTypeUint8Vector', 'DataTypeUint16Vector', 'DataTypeUint32Vector', 
        'DataTypeUint64Vector', 'DataTypeStringVector', 'DataTypeBoolVector'
    }
    
    # Separate scalar and vector columns
    scalar_fill_map = {}
    vector_fill_map = {}
    
    for col, datatype in offline_col_to_datatype_map.items():
        default_val = offline_col_to_default_values_map[col]
        if datatype in vector_types:
            default_val = default_val.strip('[]').split(',')
            if datatype in {'DataTypeFP8E5M2Vector', 'DataTypeFP8E4M3Vector', 'DataTypeFP16Vector', 'DataTypeFP32Vector', 'DataTypeFP64Vector'}:
                default_val = [float(x) for x in default_val]
            elif datatype in {'DataTypeInt8Vector', 'DataTypeInt16Vector', 'DataTypeInt32Vector', 'DataTypeInt64Vector', 
                              'DataTypeUint8Vector', 'DataTypeUint16Vector', 'DataTypeUint32Vector', 'DataTypeUint64Vector'}:
                default_val = [int(x) for x in default_val]
            elif datatype == 'DataTypeBoolVector':                
                # Assert each element is either true or false
                assert all(x.lower() in ['true', 'false'] for x in default_val), f"All elements in bool vector must be 'true' or 'false', got {default_val}"
            # String vector doesn't need casting
            vector_fill_map[col] = default_val
        else:
            if datatype == 'DataTypeBool':
                assert default_val.lower() in ['true', 'false'], f"Bool value must be 'true' or 'false', got {default_val}"
            scalar_fill_map[col] = default_val

    # Fill scalar columns with fillna
    if scalar_fill_map:
        df = df.fillna(scalar_fill_map)
    
    # Fill vector columns with withColumn/when/otherwise
    for col, default_val in vector_fill_map.items():
        df = df.withColumn(
            col,
            F.when(F.col(col).isNull(), F.array([F.lit(x) for x in default_val])).otherwise(F.col(col))
        )
    return df

def join_with_users_of_60_days(spark, df):
    
    assert "user_id" in df.columns, "user_id column not found in df"
        
    users = spark.sql("""
                    select distinct user_id 
                    from silver.mixpanel_android__app_open a 
                    WHERE concat(year,'-',month,'-',day) >= date_add(CURRENT_TIMESTAMP,-60) 
                    AND concat(year,'-',month,'-',day) <= date_add(CURRENT_TIMESTAMP,-1)
                """)
    
    df = df.join(users, "user_id")
    return df