#!/usr/bin/env python3
"""
asynclogparse - Python version using inference-logging-client
Parses asyncloguploader log files and outputs human-readable format
"""

import sys
import os
import argparse
from pathlib import Path
import json
import urllib.request
import urllib.parse

try:
    import inference_logging_client as ilc
except ImportError:
    print("Error: inference-logging-client not installed. Install with: pip install inference-logging-client")
    sys.exit(1)


def deframe_log_file(log_path):
    """
    Deframe the binary log file, removing headers and padding.
    Returns list of protobuf records.
    """
    records = []
    
    with open(log_path, 'rb') as f:
        while True:
            # Read 8-byte header: capacity (4 bytes) + validDataBytes (4 bytes)
            header = f.read(8)
            if len(header) < 8:
                break
            
            capacity = int.from_bytes(header[0:4], byteorder='little')
            valid_data_bytes = int.from_bytes(header[4:8], byteorder='little')
            
            if valid_data_bytes == 0:
                break
            
            # Read the data block
            data_block = f.read(capacity)
            if len(data_block) < valid_data_bytes:
                break
            
            # Extract valid records from the data block
            offset = 0
            while offset < valid_data_bytes:
                if offset + 4 > valid_data_bytes:
                    break
                
                # Read 4-byte length prefix
                record_length = int.from_bytes(
                    data_block[offset:offset+4], 
                    byteorder='little'
                )
                offset += 4
                
                if record_length == 0 or offset + record_length > valid_data_bytes:
                    break
                
                # Extract the record
                record = data_block[offset:offset+record_length]
                records.append(record)
                offset += record_length
                
                # Skip padding (zeros) until next 4-byte boundary or end
                while offset < valid_data_bytes and offset % 4 != 0:
                    offset += 1
    
    return records


def fetch_feature_schema(model_config_id, version=1):
    """
    Fetch feature schema from the inference API (Custodian API).
    Returns a list of FeatureInfo objects compatible with inference-logging-client.
    """
    # Cache schema per model_config_id:version
    if not hasattr(fetch_feature_schema, '_cache'):
        fetch_feature_schema._cache = {}
    
    cache_key = f"{model_config_id}:{version}"
    if cache_key in fetch_feature_schema._cache:
        return fetch_feature_schema._cache[cache_key]
    
    inference_host = os.getenv("INFERENCE_HOST", "http://custodian.prd.meesho.int")
    inference_path = os.getenv("INFERENCE_PATH", "/api/v1/custodian/mp-config-registry/get_feature_schema")
    
    # Build URL with query parameters
    params = urllib.parse.urlencode({'model_config_id': model_config_id, 'version': version})
    url = f"{inference_host}{inference_path}?{params}"
    
    try:
        # Try GET request
        req = urllib.request.Request(url)
        req.add_header('Content-Type', 'application/json')
        req.add_header('Accept', 'application/json')
        
        with urllib.request.urlopen(req, timeout=10) as response:
                    if response.status == 200:
                        schema_data = json.loads(response.read().decode('utf-8'))
                        print(f"[DEBUG] Schema data type: {type(schema_data)}", flush=True)
                        
                        # Handle dict response - extract the actual schema list
                        if isinstance(schema_data, dict):
                            # The API might return {"features": [...]} or similar
                            if 'features' in schema_data:
                                schema_list = schema_data['features']
                            elif 'schema' in schema_data:
                                schema_list = schema_data['schema']
                            elif 'data' in schema_data:
                                schema_list = schema_data['data']
                            else:
                                # Try to find any list value
                                schema_list = [v for v in schema_data.values() if isinstance(v, list)]
                                if schema_list:
                                    schema_list = schema_list[0]
                                else:
                                    print(f"[WARNING] Could not find schema list in dict: {list(schema_data.keys())}", flush=True)
                                    schema_list = []
                        elif isinstance(schema_data, list):
                            schema_list = schema_data
                        else:
                            print(f"[WARNING] Unexpected schema data type: {type(schema_data)}", flush=True)
                            schema_list = []
                        
                        print(f"[DEBUG] Schema list length: {len(schema_list)}, first item: {schema_list[0] if schema_list else 'empty'}", flush=True)
                        
                        # Convert to FeatureInfo objects
                        if hasattr(ilc, 'FeatureInfo'):
                            FeatureInfo = ilc.FeatureInfo
                            import inspect
                            try:
                                sig = inspect.signature(FeatureInfo)
                                print(f"[DEBUG] FeatureInfo signature: {sig}", flush=True)
                            except:
                                pass
                            
                            schema = []
                            for idx, item in enumerate(schema_list):
                                if isinstance(item, dict):
                                    try:
                                        # FeatureInfo needs: name, feature_type, index
                                        feat_info = FeatureInfo(
                                            name=item.get('feature_name', item.get('name', '')),
                                            feature_type=item.get('feature_type', item.get('type', '')),
                                            index=item.get('index', idx)
                                        )
                                        schema.append(feat_info)
                                        if idx == 0:
                                            print(f"[DEBUG] Created FeatureInfo: name={feat_info.name}, type={feat_info.feature_type}, index={feat_info.index}", flush=True)
                                    except Exception as e:
                                        print(f"[WARNING] Failed to create FeatureInfo from {item}: {str(e)}", flush=True)
                                        try:
                                            # Try with all dict keys
                                            feat_info = FeatureInfo(**item, index=idx)
                                            schema.append(feat_info)
                                        except Exception as e2:
                                            print(f"[WARNING] **item also failed: {str(e2)}", flush=True)
                                            schema.append(item)
                                else:
                                    schema.append(item)
                        else:
                            schema = schema_list
                        
                        print(f"[DEBUG] Final schema: {len(schema)} FeatureInfo objects", flush=True)
                        fetch_feature_schema._cache[cache_key] = schema
                        return schema
    except urllib.error.HTTPError as e:
        if e.code == 404 or e.code == 400:
            # Try POST if GET fails
            try:
                post_url = f"{inference_host}{inference_path}"
                post_data = json.dumps({"model_config_id": model_config_id, "version": version}).encode('utf-8')
                req = urllib.request.Request(post_url, data=post_data, method='POST')
                req.add_header('Content-Type', 'application/json')
                req.add_header('Accept', 'application/json')
                
                with urllib.request.urlopen(req, timeout=10) as response:
                    if response.status == 200:
                        schema_data = json.loads(response.read().decode('utf-8'))
                        # Convert to FeatureInfo objects
                        if hasattr(ilc, 'FeatureInfo'):
                            FeatureInfo = ilc.FeatureInfo
                            schema = []
                            for item in schema_data:
                                if isinstance(item, dict):
                                    schema.append(FeatureInfo(
                                        feature_name=item.get('feature_name', ''),
                                        feature_type=item.get('feature_type', '')
                                    ))
                                else:
                                    schema.append(item)
                        else:
                            schema = schema_data if isinstance(schema_data, list) else []
                        fetch_feature_schema._cache[cache_key] = schema
                        return schema
            except Exception as post_err:
                print(f"[WARNING] POST also failed for {model_config_id}:v{version}: {str(post_err)}", flush=True)
        else:
            print(f"[WARNING] GET failed for {model_config_id}:v{version}: HTTP {e.code}", flush=True)
    except Exception as e:
        print(f"[WARNING] Failed to fetch schema for {model_config_id}:v{version}: {str(e)}", flush=True)
        return None
    
    return None


def parse_and_decode_mplog(record):
    """
    Parse MPLog protobuf and decode features using inference-logging-client.
    """
    try:
        # Parse the MPLog protobuf using inference-logging-client
        mplog = ilc.parse_mplog_protobuf(record)
        
        # Extract basic fields
        user_id = getattr(mplog, 'user_id', '')
        tracking_id = getattr(mplog, 'tracking_id', '')
        mp_config_id = getattr(mplog, 'mp_config_id', '') or getattr(mplog, 'model_proxy_config_id', '')
        entities = getattr(mplog, 'entities', []) or []
        parent_entities = getattr(mplog, 'parent_entity', []) or []
        metadata = getattr(mplog, 'metadata', b'')
        
        # Check if version is present in the protobuf (field 4)
        # In proto3, we can't distinguish between "not set" and "set to default (0)"
        # So we check if the attribute exists and has a non-None value
        # If version is 0, it could be either default or explicitly set to 0
        try:
            version = getattr(mplog, 'version')
            # If version is None or not set, it will raise AttributeError
            # If version exists, check if it's a valid int (could be 0, which is valid)
            if version is None:
                version = 1
                version_present = False
            else:
                # Version exists - in proto3, we assume it was present if it's not None
                # Note: version=0 is a valid value, so we can't distinguish from "not set"
                version_present = True
        except (AttributeError, TypeError):
            version = 1
            version_present = False
        
        format_type = getattr(mplog, 'format_type', None)
        
        # Get features from the parsed MPLog
        # The structure: mplog.features is a list of perEntityFeatures objects
        # Each perEntityFeatures has: encoded_features (bytes) that need decoding
        
        # Debug: Check what attributes the mplog has
        mplog_attrs = [x for x in dir(mplog) if not x.startswith('__')]
        
        # Try to get features - check multiple possible attribute names
        # According to the protobuf schema, it could be:
        # 1. mplog.features (list of perEntityFeatures objects with encoded_features)
        # 2. mplog.encoded_features (direct list of bytes - field 8)
        features_attr = []
        encoded_features_direct = []
        
        # First, try to get features as a list of objects
        if hasattr(mplog, 'features'):
            features_attr = getattr(mplog, 'features', []) or []
        elif hasattr(mplog, '_features'):
            features_attr = getattr(mplog, '_features', []) or []
        elif hasattr(mplog, 'per_entity_features'):
            features_attr = getattr(mplog, 'per_entity_features', []) or []
        
        # Also check for direct encoded_features list (field 8 in protobuf)
        if hasattr(mplog, 'encoded_features'):
            encoded_features_direct = getattr(mplog, 'encoded_features', []) or []
        elif hasattr(mplog, '_encoded_features'):
            encoded_features_direct = getattr(mplog, '_encoded_features', []) or []
        
        # If we have direct encoded_features but no features_attr, use the direct list
        if not features_attr and encoded_features_direct:
            # Convert direct bytes list to feature objects structure
            features_attr = encoded_features_direct
        
        encoded_features_list = []  # Track encoded features for return value
        
        # Decode features for each entity using inference-logging-client
        decoded_entities = []
        print(f"[DEBUG] Processing {len(features_attr)} feature objects for {len(entities)} entities", flush=True)
        for i, feature_obj in enumerate(features_attr):
            if i % 50 == 0 or i < 5:  # Log first 5 and then every 50th
                print(f"[DEBUG] Processing entity {i+1}/{len(features_attr)}", flush=True)
            entity_id = entities[i] if i < len(entities) else ''
            parent_entity = parent_entities[i] if i < len(parent_entities) else ''
            
            # Get encoded_features from the feature object
            # It could be:
            # 1. A perEntityFeatures object with .encoded_features attribute
            # 2. A direct bytes object (if encoded_features is a direct list)
            encoded_features = None
            if isinstance(feature_obj, bytes):
                # Direct bytes - this is the encoded_features itself
                encoded_features = feature_obj
            elif hasattr(feature_obj, 'encoded_features'):
                encoded_features = getattr(feature_obj, 'encoded_features', b'')
            else:
                # Debug: Check what attributes the feature object has
                if hasattr(feature_obj, '__dict__'):
                    feat_attrs = feature_obj.__dict__
                    # Try common attribute names
                    if '_encoded_features' in feat_attrs:
                        encoded_features = feat_attrs['_encoded_features']
                    elif 'data' in feat_attrs:
                        encoded_features = feat_attrs['data']
                    elif 'value' in feat_attrs:
                        encoded_features = feat_attrs['value']
                    elif 'encoded_features' in feat_attrs:
                        encoded_features = feat_attrs['encoded_features']
            
            # Track encoded features for return value
            encoded_features_list.append(encoded_features if encoded_features else b'')
            
            decoded_features = {}
            if encoded_features and len(encoded_features) > 0:
                if i < 3:  # Only log for first few entities
                    print(f"[DEBUG] Decoding features for entity {i+1}, encoded_features length: {len(encoded_features)}", flush=True)
                try:
                    # decode_proto_features takes 2 arguments: (encoded_features, schema)
                    # We need to fetch the schema first from the inference API
                    
                    # Fetch schema (cached per model_config_id:version)
                    if i == 0:
                        print(f"[DEBUG] Fetching schema for {mp_config_id}:v{version}...", flush=True)
                    schema = fetch_feature_schema(mp_config_id, version)
                    
                    if schema:
                        if i == 0:
                            print(f"[DEBUG] Got schema with {len(schema)} features, decoding...", flush=True)
                            if len(schema) > 0:
                                print(f"[DEBUG] First schema item type: {type(schema[0])}, value: {schema[0]}", flush=True)
                        # decode_proto_features(encoded_features, schema)
                        decoded_features = ilc.decode_proto_features(encoded_features, schema)
                        if i < 3:
                            print(f"[DEBUG] Successfully decoded entity {i+1} features", flush=True)
                            if isinstance(decoded_features, dict):
                                print(f"[DEBUG] Decoded features keys: {list(decoded_features.keys())[:10]}", flush=True)
                    else:
                        decoded_features = {
                            '_note': 'Features need decoding with schema',
                            '_error': 'Failed to fetch schema from inference API',
                            '_encoded_length': len(encoded_features)
                        }
                except Exception as e:
                    # Other error (maybe wrong parameters, network issue, etc.)
                    if i < 3:
                        print(f"[DEBUG] Exception during decode: {str(e)[:200]}", flush=True)
                    decoded_features = {
                        '_decode_error': str(e)[:200],  # Truncate long errors
                        '_error_type': type(e).__name__
                    }
            else:
                decoded_features = {'_note': 'No encoded_features found in feature object'}
            
            decoded_entities.append({
                'entity_id': entity_id,
                'parent_entity': parent_entity,
                'features': decoded_features,
                'encoded_features_length': len(encoded_features) if encoded_features else 0
            })
        
        # Debug info if no features found
        debug_info = {}
        if len(features_attr) == 0:
            debug_info = {
                '_debug': 'No features found in mplog',
                '_mplog_attrs': mplog_attrs[:20] if 'mplog_attrs' in locals() else [],
                '_features_attr_type': type(features_attr).__name__,
                '_entities_count': len(entities)
            }
        
        return {
            'user_id': user_id,
            'tracking_id': tracking_id,
            'mp_config_id': mp_config_id,
            'entities': entities,
            'parent_entities': parent_entities,
            'metadata': metadata,
            'version': version,
            'version_present': version_present,
            'format_type': format_type,
            'decoded_entities': decoded_entities,
            'num_encoded_features': len(encoded_features_list),
            **debug_info
        }
    
    except Exception as e:
        import traceback
        return {'_parse_error': f'{str(e)}\n{traceback.format_exc()}'}


def format_output(parsed_data, output_file, record_num):
    """
    Format and write parsed data to output file.
    """
    if '_parse_error' in parsed_data:
        output_file.write(f"=== Record {record_num}: PARSE ERROR ===\n")
        output_file.write(f"Error: {parsed_data['_parse_error']}\n\n")
        return
    
    output_file.write(f"=== Record {record_num} ===\n")
    output_file.write(f"User ID: {parsed_data['user_id']}\n")
    output_file.write(f"Tracking ID: {parsed_data['tracking_id']}\n")
    output_file.write(f"Model Config ID: {parsed_data['mp_config_id']}\n")
    
    # Schema version information
    version = parsed_data.get('version', 1)
    version_present = parsed_data.get('version_present', False)
    output_file.write(f"Schema Version: {version}")
    if version_present:
        output_file.write(" (present in log)\n")
    else:
        output_file.write(" (defaulted, not present in log)\n")
    
    # Format type information
    format_type = parsed_data.get('format_type')
    if format_type is not None:
        format_names = {0: 'proto', 1: 'arrow', 2: 'parquet'}
        format_name = format_names.get(format_type, f'unknown({format_type})')
        output_file.write(f"Format Type: {format_type} ({format_name})\n")
    
    output_file.write(f"Entities: {len(parsed_data['entities'])}\n")
    if parsed_data['entities']:
        output_file.write(f"Entity IDs: {parsed_data['entities']}\n")
    output_file.write(f"Parent Entities: {len(parsed_data['parent_entities'])}\n")
    if parsed_data['parent_entities']:
        output_file.write(f"Parent Entity IDs: {parsed_data['parent_entities']}\n")
    output_file.write(f"Encoded Features: {parsed_data['num_encoded_features']}\n")
    if parsed_data['metadata']:
        output_file.write(f"Metadata: {len(parsed_data['metadata'])} bytes\n")
    
    # Show debug info if no features found
    if '_debug' in parsed_data:
        output_file.write(f"\n[DEBUG] {parsed_data.get('_debug', '')}\n")
        if '_mplog_attrs' in parsed_data and parsed_data['_mplog_attrs']:
            output_file.write(f"[DEBUG] MPLog attributes (first 20): {parsed_data['_mplog_attrs']}\n")
        if '_entities_count' in parsed_data:
            output_file.write(f"[DEBUG] Entities count: {parsed_data['_entities_count']}\n")
    
    output_file.write("\n")
    
    # Write decoded entities and features
    for i, entity_data in enumerate(parsed_data['decoded_entities']):
        output_file.write(f"--- Entity {i+1} ---\n")
        if entity_data['entity_id']:
            output_file.write(f"Entity ID: {entity_data['entity_id']}\n")
        if entity_data['parent_entity']:
            output_file.write(f"Parent Entity: {entity_data['parent_entity']}\n")
        
        output_file.write("Features:\n")
        if entity_data['features']:
            for feat_name, feat_value in entity_data['features'].items():
                output_file.write(f"  {feat_name}: {feat_value}\n")
        else:
            output_file.write(f"  (encoded_features length: {entity_data['encoded_features_length']} bytes)\n")
        output_file.write("\n")
    
    output_file.write("\n")


def main():
    parser = argparse.ArgumentParser(
        description='Parse asyncloguploader log files using inference-logging-client'
    )
    parser.add_argument('log_file', help='Path to the log file')
    parser.add_argument('-o', '--output', default=None, 
                       help='Output file path (default: <log_file>.parsed.log)')
    
    args = parser.parse_args()
    
    log_path = Path(args.log_file)
    if not log_path.exists():
        print(f"Error: Log file not found: {log_path}")
        sys.exit(1)
    
    if args.output:
        output_path = Path(args.output)
    else:
        output_path = log_path.with_suffix('.parsed.log')
    
    print(f"Parsing log file: {log_path}")
    print(f"Writing output to: {output_path}")
    
    # Deframe the log file
    print("Deframing log file...")
    records = deframe_log_file(log_path)
    print(f"Found {len(records)} records")
    
    if not records:
        print("No records found in log file.")
        return
    
    # Process all records
    print(f"Processing {len(records)} records...")
    with open(output_path, 'w') as out_file:
        for i, record in enumerate(records):
            print(f"[DEBUG] Starting record {i+1}/{len(records)}", flush=True)
            parsed_data = parse_and_decode_mplog(record)
            print(f"[DEBUG] Parsed record {i+1}, formatting output...", flush=True)
            format_output(parsed_data, out_file, i+1)
            out_file.flush()  # Ensure output is written immediately
            print(f"[DEBUG] Completed record {i+1}", flush=True)
            
            if (i + 1) % 100 == 0:
                print(f"Processed {i+1}/{len(records)} records...")
    
    print(f"Parsing complete. Output written to: {output_path}")


if __name__ == '__main__':
    main()
