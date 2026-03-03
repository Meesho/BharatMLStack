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
import struct

try:
    import inference_logging_client as ilc
except ImportError:
    print("Error: inference-logging-client not installed. Install with: pip install inference-logging-client")
    sys.exit(1)


# Constants for the log file format
HEADER_SIZE = 8  # 4 bytes capacity + 4 bytes validDataBytes
LENGTH_PREFIX_SIZE = 4  # 4 bytes for record length


def deframe_log_file(log_path):
    """
    Deframe the binary log file, removing headers and padding.
    Returns list of protobuf records.
    """
    records = []
    frame_count = 0
    skipped_frames = 0
    
    with open(log_path, 'rb') as f:
        while True:
            # Read 8-byte header: capacity (4 bytes) + validDataBytes (4 bytes)
            frame_start_pos = f.tell()
            header = f.read(8)
            if len(header) < 8:
                # End of file
                break
            
            capacity = int.from_bytes(header[0:4], byteorder='little')
            valid_data_bytes = int.from_bytes(header[4:8], byteorder='little')
            
            frame_count += 1
            
            # Debug first few frames - show raw header bytes
            if frame_count <= 5:
                header_hex = ' '.join(f'{b:02x}' for b in header)
                print(f"[DEBUG] Frame {frame_count} @ pos {frame_start_pos}: header_bytes=[{header_hex}]", flush=True)
                print(f"[DEBUG]   capacity={capacity} ({capacity/(1024*1024):.2f}MB), valid_data={valid_data_bytes} ({valid_data_bytes/(1024*1024):.2f}MB)", flush=True)
                # Check if valid_data_bytes > capacity (invalid)
                if valid_data_bytes > capacity:
                    print(f"[ERROR]   INVALID: valid_data_bytes ({valid_data_bytes}) > capacity ({capacity})!", flush=True)
            
            # Skip frames with 0 valid data bytes, but continue processing
            # (there might be more frames after empty ones)
            if valid_data_bytes == 0:
                skipped_frames += 1
                # Still need to read the remaining capacity bytes to maintain file position
                # Frame is capacity bytes total, we already read 8 bytes (header), so read capacity-8
                if capacity > 8:
                    f.read(capacity - 8)
                continue
            
            # Validate capacity - check for reasonable values
            # Capacity should be at least 8 (header size)
            if capacity < 8:
                print(f"[WARNING] Invalid frame capacity {capacity} at frame {frame_count}, skipping", flush=True)
                continue
            
            # Check if capacity is reasonable given remaining file size
            # This helps detect file position misalignment
            # Note: Frame is capacity bytes total, we already read 8 bytes (header)
            current_pos = f.tell()
            f.seek(0, 2)  # Seek to end to get file size
            file_size = f.tell()
            f.seek(current_pos)  # Restore position
            remaining = file_size - current_pos
            
            # Validate capacity and valid_data_bytes are reasonable
            # valid_data_bytes is the data size (excluding header), so it should be <= capacity - 8
            # But we check valid_data_bytes <= capacity for basic validation
            if valid_data_bytes > capacity:
                print(f"[ERROR] Frame {frame_count}: valid_data_bytes ({valid_data_bytes}) > capacity ({capacity}). Header appears corrupted.", flush=True)
                print(f"[ERROR]   Attempting to recover by skipping this frame and trying next position...", flush=True)
                
                # Try to find next valid frame by scanning forward
                # Look for a pattern that might be a valid header (capacity >= 8, valid_data <= capacity)
                # Try alignment boundaries first (4KB = 4096 bytes), then byte-by-byte
                recovery_attempts = 0
                max_recovery_attempts = 10000  # Scan up to 10KB forward
                found_valid_frame = False
                alignment_size = 4096  # 4KB alignment
                
                # First, try 4KB-aligned positions (common in Direct I/O)
                for aligned_offset in range(alignment_size, max_recovery_attempts, alignment_size):
                    test_pos = frame_start_pos + aligned_offset
                    f.seek(test_pos)
                    test_header = f.read(8)
                    
                    if len(test_header) < 8:
                        break
                    
                    test_capacity = int.from_bytes(test_header[0:4], byteorder='little')
                    test_valid_data = int.from_bytes(test_header[4:8], byteorder='little')
                    
                    # Check if this looks like a valid header
                    # Also check if valid_data is reasonable (not too small, not too large)
                    if (test_capacity >= 8 and test_capacity < 500 * 1024 * 1024 and 
                        test_valid_data <= test_capacity and test_valid_data > 0 and
                        test_valid_data >= 4):  # At least 4 bytes for a length prefix
                        # Verify by trying to read a sample of the data and check for valid record structure
                        # Read first few bytes to see if they look like record length prefixes
                        f.seek(test_pos + 8)  # Skip header
                        sample_data = f.read(min(100, test_valid_data))  # Read first 100 bytes or valid_data, whichever is smaller
                        f.seek(test_pos)  # Restore position
                        
                        # Check if first few bytes look like reasonable record lengths
                        # Record lengths should be reasonable (not millions of bytes for a single record)
                        looks_valid = True
                        if len(sample_data) >= 4:
                            first_record_len = int.from_bytes(sample_data[0:4], byteorder='little')
                            # Record length should be reasonable (between 4 bytes and 10MB)
                            if first_record_len > 10 * 1024 * 1024 or first_record_len == 0:
                                looks_valid = False
                        
                        if looks_valid:
                            # Found a potential valid frame
                            print(f"[INFO] Found potential valid frame at 4KB-aligned offset +{aligned_offset} bytes", flush=True)
                            f.seek(test_pos)
                            header = test_header
                            capacity = test_capacity
                            valid_data_bytes = test_valid_data
                            frame_start_pos = test_pos
                            found_valid_frame = True
                            break
                
                # If not found at aligned positions, try byte-by-byte (slower but more thorough)
                if not found_valid_frame:
                    print(f"[INFO] No valid frame at 4KB boundaries, trying byte-by-byte scan...", flush=True)
                    for recovery_attempts in range(1, max_recovery_attempts):
                        test_pos = frame_start_pos + recovery_attempts
                        f.seek(test_pos)
                        test_header = f.read(8)
                        
                        if len(test_header) < 8:
                            break
                        
                        test_capacity = int.from_bytes(test_header[0:4], byteorder='little')
                        test_valid_data = int.from_bytes(test_header[4:8], byteorder='little')
                        
                        # Check if this looks like a valid header
                        # Also check if valid_data is reasonable
                        if (test_capacity >= 8 and test_capacity < 500 * 1024 * 1024 and 
                            test_valid_data <= test_capacity and test_valid_data > 0 and
                            test_valid_data >= 4):  # At least 4 bytes for a length prefix
                            # Verify by trying to read a sample of the data
                            f.seek(test_pos + 8)  # Skip header
                            sample_data = f.read(min(100, test_valid_data))
                            f.seek(test_pos)  # Restore position
                            
                            # Check if first few bytes look like reasonable record lengths
                            looks_valid = True
                            if len(sample_data) >= 4:
                                first_record_len = int.from_bytes(sample_data[0:4], byteorder='little')
                                # Record length should be reasonable (between 4 bytes and 10MB)
                                if first_record_len > 10 * 1024 * 1024 or first_record_len == 0:
                                    looks_valid = False
                            
                            if looks_valid:
                                # Found a potential valid frame
                                print(f"[INFO] Found potential valid frame at offset +{recovery_attempts} bytes", flush=True)
                                f.seek(test_pos)
                                header = test_header
                                capacity = test_capacity
                                valid_data_bytes = test_valid_data
                                frame_start_pos = test_pos
                                found_valid_frame = True
                                break
                
                if not found_valid_frame:
                    print(f"[ERROR] Could not find valid frame after {recovery_attempts} attempts.", flush=True)
                    print(f"[ERROR] Frame 2 appears corrupted. Attempting to continue from Frame 1's end position...", flush=True)
                    # Try to continue from where Frame 1 ended (position 8388616)
                    # Maybe Frame 2's header is completely wrong, so skip it entirely
                    # and try to find the next frame by scanning from Frame 1's end
                    f.seek(8388616)  # Position after Frame 1
                    # Skip the corrupted header bytes and try to find next valid frame
                    # Scan forward more aggressively
                    scan_start = 8388616
                    max_scan = 100000  # Scan up to 100KB forward
                    found_next_frame = False
                    
                    for scan_offset in range(0, max_scan, 4096):  # Try 4KB aligned positions first
                        test_pos = scan_start + scan_offset
                        f.seek(test_pos)
                        test_header = f.read(8)
                        
                        if len(test_header) < 8:
                            break
                        
                        test_capacity = int.from_bytes(test_header[0:4], byteorder='little')
                        test_valid_data = int.from_bytes(test_header[4:8], byteorder='little')
                        
                        # Stricter validation: capacity must be reasonable and valid_data must be reasonable
                        if (test_capacity >= 8 and test_capacity < 100 * 1024 * 1024 and  # Max 100MB capacity
                            test_valid_data > 0 and test_valid_data <= test_capacity and
                            test_valid_data >= 4):
                            # Check remaining file size
                            # Frame is test_capacity bytes total, we need test_capacity-8 bytes for data section
                            current_pos = f.tell()
                            f.seek(0, 2)
                            file_size = f.tell()
                            f.seek(current_pos)
                            remaining = file_size - test_pos
                            
                            if test_capacity - 8 <= remaining:
                                # Verify by checking first record
                                f.seek(test_pos + 8)
                                sample = f.read(min(100, test_valid_data))
                                f.seek(test_pos)
                                
                                if len(sample) >= 4:
                                    first_len = int.from_bytes(sample[0:4], byteorder='little')
                                    if 4 <= first_len <= 10 * 1024 * 1024:
                                        print(f"[INFO] Found valid frame at position {test_pos} (offset +{scan_offset} from Frame 1 end)", flush=True)
                                        f.seek(test_pos)
                                        header = test_header
                                        capacity = test_capacity
                                        valid_data_bytes = test_valid_data
                                        frame_start_pos = test_pos
                                        found_next_frame = True
                                        break
                    
                    if not found_next_frame:
                        print(f"[ERROR] Could not find next valid frame. File may be corrupted or incomplete.", flush=True)
                        break
                else:
                    print(f"[INFO] Continuing with recovered frame at position {frame_start_pos}", flush=True)
            
            # Check if we have enough remaining file for the data section (capacity - 8 bytes)
            data_size_needed = capacity - 8
            if data_size_needed > remaining:
                print(f"[WARNING] Frame {frame_count}: data section needs {data_size_needed} bytes (capacity {capacity} - 8 header) but only {remaining} bytes remaining.", flush=True)
                print(f"[DEBUG] Frame start position was {frame_start_pos}, current position is {current_pos}", flush=True)
                
                # If this frame was recovered (frame_count == 2 and we had a corrupted header), try fallback recovery
                if frame_count == 2 and frame_start_pos > 8388616:
                    print(f"[INFO] Recovered frame failed validation. Trying fallback recovery from Frame 1's end position...", flush=True)
                    # Fallback: scan from Frame 1's end position (8388616)
                    f.seek(8388616)
                    max_scan = 10 * 1024 * 1024  # Scan up to 10MB forward (more aggressive)
                    found_next_frame = False
                    
                    # Strategy 1: Try 4KB aligned positions first (faster)
                    print(f"[INFO] Scanning 4KB-aligned positions from Frame 1 end (up to 1MB)...", flush=True)
                    for scan_offset in range(0, min(max_scan, 1000000), 4096):  # Up to 1MB with 4KB alignment
                        test_pos = 8388616 + scan_offset
                        f.seek(test_pos)
                        test_header = f.read(8)
                        
                        if len(test_header) < 8:
                            break
                        
                        test_capacity = int.from_bytes(test_header[0:4], byteorder='little')
                        test_valid_data = int.from_bytes(test_header[4:8], byteorder='little')
                        
                        # Stricter validation
                        if (test_capacity >= 8 and test_capacity < 100 * 1024 * 1024 and  # Max 100MB
                            test_valid_data > 0 and test_valid_data <= test_capacity and
                            test_valid_data >= 4):
                            # Check remaining file size
                            # Frame is test_capacity bytes total, we need test_capacity-8 bytes for data section
                            current_test_pos = f.tell()
                            f.seek(0, 2)
                            file_size = f.tell()
                            f.seek(current_test_pos)
                            remaining_test = file_size - test_pos
                            
                            if test_capacity - 8 <= remaining_test:
                                # Verify by checking first record
                                f.seek(test_pos + 8)
                                sample = f.read(min(100, test_valid_data))
                                f.seek(test_pos)
                                
                                if len(sample) >= 4:
                                    first_len = int.from_bytes(sample[0:4], byteorder='little')
                                    if 4 <= first_len <= 10 * 1024 * 1024:
                                        print(f"[INFO] Found valid frame at position {test_pos} (offset +{scan_offset} from Frame 1 end)", flush=True)
                                        f.seek(test_pos)
                                        header = test_header
                                        capacity = test_capacity
                                        valid_data_bytes = test_valid_data
                                        frame_start_pos = test_pos
                                        found_next_frame = True
                                        break
                    
                    # Strategy 2: If not found, try byte-by-byte (slower but more thorough)
                    if not found_next_frame:
                        print(f"[INFO] No frame found at 4KB boundaries, trying byte-by-byte scan (up to 100KB)...", flush=True)
                        for scan_offset in range(0, min(max_scan, 100000), 1):  # Up to 100KB byte-by-byte
                            test_pos = 8388616 + scan_offset
                            f.seek(test_pos)
                            test_header = f.read(8)
                            
                            if len(test_header) < 8:
                                break
                            
                            test_capacity = int.from_bytes(test_header[0:4], byteorder='little')
                            test_valid_data = int.from_bytes(test_header[4:8], byteorder='little')
                            
                            # Stricter validation
                            if (test_capacity >= 8 and test_capacity < 100 * 1024 * 1024 and
                                test_valid_data > 0 and test_valid_data <= test_capacity and
                                test_valid_data >= 4):
                                # Check remaining file size
                                # Frame is test_capacity bytes total, we need test_capacity-8 bytes for data section
                                current_test_pos = f.tell()
                                f.seek(0, 2)
                                file_size = f.tell()
                                f.seek(current_test_pos)
                                remaining_test = file_size - test_pos
                                
                                if test_capacity - 8 <= remaining_test:
                                    # Verify by checking first record
                                    f.seek(test_pos + 8)
                                    sample = f.read(min(100, test_valid_data))
                                    f.seek(test_pos)
                                    
                                    if len(sample) >= 4:
                                        first_len = int.from_bytes(sample[0:4], byteorder='little')
                                        if 4 <= first_len <= 10 * 1024 * 1024:
                                            print(f"[INFO] Found valid frame at position {test_pos} (offset +{scan_offset} from Frame 1 end)", flush=True)
                                            f.seek(test_pos)
                                            header = test_header
                                            capacity = test_capacity
                                            valid_data_bytes = test_valid_data
                                            frame_start_pos = test_pos
                                            found_next_frame = True
                                            break
                    
                    if found_next_frame:
                        # Continue with the found frame
                        print(f"[INFO] Continuing with fallback-recovered frame at position {frame_start_pos}", flush=True)
                        continue
                    else:
                        # Get file size for error message
                        f.seek(0, 2)
                        file_size = f.tell()
                        print(f"[ERROR] Fallback recovery failed after scanning {min(max_scan, 100000)} bytes.", flush=True)
                        print(f"[ERROR] File size: {file_size} bytes. Scanned from position 8388616 to {8388616 + min(max_scan, 100000)}.", flush=True)
                        print(f"[ERROR] File may be corrupted, incomplete, or have only 1 record.", flush=True)
                        break
                else:
                    # Not a recovered frame, or not frame 2 - just stop
                    print(f"[ERROR] Stopping due to capacity exceeding remaining file size.", flush=True)
                    break
            
            # Warn about very large capacities but continue if reasonable
            if capacity > 100 * 1024 * 1024:  # 100MB
                print(f"[INFO] Frame {frame_count} has large capacity {capacity/(1024*1024):.1f}MB (remaining file: {remaining/(1024*1024):.1f}MB). Continuing...", flush=True)
            
            # Read the data block
            # IMPORTANT: Frame is capacity bytes total, which INCLUDES the 8-byte header we already read
            # So we need to read capacity - 8 bytes for the data section
            data_size = capacity - 8
            if data_size < 0:
                print(f"[WARNING] Frame {frame_count}: capacity {capacity} is too small (less than header size 8), skipping", flush=True)
                continue
            
            read_start_pos = f.tell()
            data_block = f.read(data_size)
            read_end_pos = f.tell()
            
            if frame_count <= 3:
                print(f"[DEBUG]   Reading data: start_pos={read_start_pos}, data_size={data_size} (capacity={capacity} - 8 header), read={len(data_block)} bytes, end_pos={read_end_pos}", flush=True)
            
            if len(data_block) < data_size:
                # Partial read - end of file
                print(f"[WARNING] Partial frame read at frame {frame_count}, expected {data_size} bytes, got {len(data_block)}. End of file?", flush=True)
                if len(data_block) == 0:
                    break
                # If we got partial data but it's less than valid_data_bytes, skip this frame
                if len(data_block) < valid_data_bytes:
                    print(f"[WARNING] Partial data block smaller than valid_data_bytes, skipping frame", flush=True)
                    # File position is already at EOF, so break
                    break
            
            # Check if we got enough data for valid_data_bytes
            # Note: data_block might be larger than valid_data_bytes (due to padding/zero-fill)
            # But valid_data_bytes is the actual data written (excluding the 8-byte header offset)
            # The data_block starts at byte 8 of the frame, so valid_data_bytes should be <= data_size
            if valid_data_bytes > data_size:
                print(f"[WARNING] valid_data_bytes ({valid_data_bytes}) > data_size ({data_size}) at frame {frame_count}, using data_size", flush=True)
                valid_data_bytes = data_size
            
            if len(data_block) < valid_data_bytes:
                print(f"[WARNING] Data block smaller than valid_data_bytes at frame {frame_count} (got {len(data_block)}, need {valid_data_bytes}), skipping frame", flush=True)
                # File position is already correct for next frame (we read data_size bytes)
                continue
            
            # Extract valid records from the data block
            offset = 0
            records_in_frame = 0
            while offset < valid_data_bytes:
                if offset + 4 > valid_data_bytes:
                    # Not enough bytes for length prefix
                    break
                
                # Read 4-byte length prefix
                record_length = int.from_bytes(
                    data_block[offset:offset+4], 
                    byteorder='little'
                )
                offset += 4
                
                # Skip 0-length records (padding), but continue processing
                if record_length == 0:
                    # Skip padding until next 4-byte boundary
                    while offset < valid_data_bytes and offset % 4 != 0:
                        offset += 1
                    continue
                
                # Check if record fits in remaining data
                if offset + record_length > valid_data_bytes:
                    # Record extends beyond valid data, stop processing this frame
                    break
                
                # Extract the record
                record = data_block[offset:offset+record_length]
                records.append(record)
                records_in_frame += 1
                offset += record_length
                
                # Skip padding (zeros) until next 4-byte boundary or end
                while offset < valid_data_bytes and offset % 4 != 0:
                    offset += 1
            
            if records_in_frame > 0:
                if frame_count <= 5 or frame_count % 1000 == 0:
                    print(f"[DEBUG] Frame {frame_count}: found {records_in_frame} records (total: {len(records)})", flush=True)
            elif frame_count <= 10:
                print(f"[DEBUG] Frame {frame_count}: no records found (capacity={capacity}, valid_data={valid_data_bytes})", flush=True)
    
    print(f"[DEBUG] Deframing complete: {frame_count} frames processed, {skipped_frames} empty frames skipped, {len(records)} total records", flush=True)
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
    
    # Save raw records AND full frame data for verification
    raw_records_file = log_path.with_suffix('.raw_records.bin')
    raw_frames_file = log_path.with_suffix('.raw_frames.bin')
    
    if len(records) > 0:
        print(f"Saving {len(records)} raw protobuf records to {raw_records_file} for verification...")
        with open(raw_records_file, 'wb') as f:
            for i, record in enumerate(records):
                # Write record with length prefix for easy reading
                f.write(struct.pack('<I', len(record)))
                f.write(record)
        print(f"Raw records saved. File size: {raw_records_file.stat().st_size:,} bytes")
        
        # Also save full frame data (for exact reconstruction)
        print(f"Saving full frame data to {raw_frames_file} for exact reconstruction...")
        with open(raw_frames_file, 'wb') as f:
            # Re-read the file and save complete frame data
            with open(log_path, 'rb') as log_f:
                while True:
                    header = log_f.read(HEADER_SIZE)
                    if len(header) < HEADER_SIZE:
                        break
                    capacity = struct.unpack('<I', header[0:4])[0]
                    valid_data = struct.unpack('<I', header[4:8])[0]
                    
                    # Write frame header
                    f.write(header)
                    
                    # Write complete data section (capacity - HEADER_SIZE bytes)
                    if capacity > HEADER_SIZE:
                        data_section = log_f.read(capacity - HEADER_SIZE)
                        f.write(data_section)
                    elif capacity > 0:
                        data_section = log_f.read(capacity)
                        f.write(data_section)
        print(f"Full frame data saved. File size: {raw_frames_file.stat().st_size:,} bytes")
    
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
