import re
import sys
import requests

# Import shared utilities from bharatml_common
try:
    from bharatml_commons.http_client import get_metadata_host_response as shared_get_metadata_host_response
    from bharatml_commons.column_utils import clean_column_name as shared_clean_column_name, generate_renamed_column
    from bharatml_commons.feature_utils import get_fgs_to_feature_mappings as shared_get_fgs_to_feature_mappings, extract_entity_info
    _USE_SHARED_UTILS = True
except ImportError:
    # Keep fallback for backward compatibility during transition
    _USE_SHARED_UTILS = False

def clean_column_name(column_name):
    """
    Cleans a column name to be Delta Table compatible:
    - Replaces unsupported special characters with "_"
    - Ensures it doesn't start with a number
    - Converts to lowercase
    """
    if _USE_SHARED_UTILS:
        return shared_clean_column_name(column_name)
    
    # Fallback to local implementation
    # Define a regex pattern to match invalid characters
    invalid_chars = r"[^a-zA-Z0-9_]"  
    cleaned_name = re.sub(invalid_chars, "_", column_name)  # Replace invalid chars with "_"

    # Ensure it doesn't start with a number
    if cleaned_name[0].isdigit():
        cleaned_name = "_" + cleaned_name

    return cleaned_name.lower()  # Convert to lowercase (Delta standard)

def get_metadata_host_response(features_metadata_url: str, job_id: str, job_token: str) -> dict:
    if _USE_SHARED_UTILS:
        return shared_get_metadata_host_response(features_metadata_url, job_id, job_token)
    
    # Fallback to local implementation
    assert features_metadata_url.startswith("http") or features_metadata_url.startswith("https"), "features_metadata_url must start with http or https"
    
    params = {
        "jobId": job_id,
        "jobToken": job_token
    }
    headers = {
        "Content-Type": "application/json"
    }

    response = requests.post(features_metadata_url, headers=headers, params=params)

    assert response.status_code == 200, f"Request failed with status code {response.status_code}: {response.json()} while calling {features_metadata_url} for job_id: {job_id} and job_token: {job_token}"

    return response.json()

def get_features_from_response_v2(json_response, fg_to_consider):
    if _USE_SHARED_UTILS:
        entity_label, entity_column_names = extract_entity_info(json_response)
    else:
        entity_label = list(json_response["keys"].keys())[0]
        entity_column_names = list(json_response["keys"].values())[0]
    
    skip_check = True if not fg_to_consider else False
    fg_to_datatype_map = {}

    responses = json_response["data"]
    feature_list = []
    offline_col_to_default_values_map = {}
    offline_col_to_datatype_map = {}
    feature_kv = {}
    for sources in responses:

        source_type = sources["storage-provider"]

        base_path_config = sources['base-path']


        for table in base_path_config:

            table_name = table["source-base-path"]
            

            features_list_temp = []
            for features in table["data-paths"]:
                fg_label = features["feature-group-label"]
                fg_data_type = features["data-type"]
                
                if fg_label not in fg_to_datatype_map:
                    fg_to_datatype_map[fg_label] = fg_data_type

                if not skip_check and fg_label not in fg_to_consider:
                    continue

                feature_col = features["source-data-column"]
                feature_label = features["feature-label"]
                
                # Use shared utility if available, otherwise fallback to original logic
                if _USE_SHARED_UTILS:
                    try:
                        rename_feature_col = generate_renamed_column(table_name, source_type, feature_col)
                    except ValueError:
                        print(f"source: {table_name} of type {source_type} not expected")
                        sys.exit(1)
                else:
                    # Original logic
                    if source_type == "TABLE":
                        rename_feature_col = table_name.split(".")[1] + "___" + feature_col
                                            
                    elif source_type in ["PARQUET_GCS", # GCP
                                         "PARQUET_S3", # AWS
                                         "PARQUET_ADLS", # Azure
                                         "DELTA_GCS", # GCP
                                         "DELTA_S3", # AWS
                                         "DELTA_ADLS" # Azure
                                         ]:
                        rename_feature_col = clean_column_name(table_name.split("gs://")[1].strip("/ ").split("/")[-1]) + "___" + feature_col
                        
                    else:
                        print(f"source: {table_name} of type {source_type} not expected")
                        sys.exit(1)
                
                feature_default = features["default-value"]

                features_list_temp.append((feature_col, rename_feature_col))

                offline_col_to_default_values_map[rename_feature_col] = feature_default
                offline_col_to_datatype_map[rename_feature_col] = fg_data_type


                if fg_label not in feature_kv:
                    feature_kv[fg_label] = [(rename_feature_col, feature_label)]

                else:
                    feature_kv[fg_label].append((rename_feature_col, feature_label))

            if len(features_list_temp) != 0:
                feature_list.append([table_name, source_type, features_list_temp])
                
    if _USE_SHARED_UTILS:
        onfs_fg_to_onfs_feat_map, onfs_fg_to_ofs_feat_map = shared_get_fgs_to_feature_mappings(feature_kv)
    else:
        onfs_fg_to_onfs_feat_map, onfs_fg_to_ofs_feat_map = get_fgs_to_feature_mappings(feature_kv)
        
    return feature_list, offline_col_to_default_values_map, onfs_fg_to_onfs_feat_map, onfs_fg_to_ofs_feat_map, fg_to_datatype_map, entity_label, entity_column_names, offline_col_to_datatype_map

def get_features_details(features_metadata_url: str, job_id: str, job_token: str, fgs_to_consider: list = []) -> tuple:
    
    custodian_response = get_metadata_host_response(features_metadata_url, job_id, job_token)
    return get_features_from_response_v2(custodian_response, fgs_to_consider)

def get_fgs_to_feature_mappings(feature_group_kv):
    if _USE_SHARED_UTILS:
        return shared_get_fgs_to_feature_mappings(feature_group_kv)

    # Fallback to local implementation
    onfs_fg_to_onfs_feat_map = {} # onfs fg -> online feat name
    for fg in feature_group_kv:
        if fg not in onfs_fg_to_onfs_feat_map:
            onfs_fg_to_onfs_feat_map[fg] = []

        for onfs_feat_name in feature_group_kv[fg]:
            onfs_fg_to_onfs_feat_map[fg].append(onfs_feat_name[1])

    onfs_fg_to_ofs_feat_map = {} # onfs fg -> offline feat name

    for fg in feature_group_kv:
        if fg not in onfs_fg_to_ofs_feat_map:
            onfs_fg_to_ofs_feat_map[fg] = []

        for onfs_feat_name in feature_group_kv[fg]:
            onfs_fg_to_ofs_feat_map[fg].append(onfs_feat_name[0])       
            
    return onfs_fg_to_onfs_feat_map, onfs_fg_to_ofs_feat_map