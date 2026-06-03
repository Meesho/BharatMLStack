#!/usr/bin/env python3
"""Quick verification of parsing completeness"""
import struct
import os

log_file = 'Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log'
parsed_file = 'Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.parsed.log'

# File sizes
log_size = os.path.getsize(log_file)
parsed_size = os.path.getsize(parsed_file)
print(f"Original log: {log_size:,} bytes ({log_size/(1024*1024):.2f} MB)")
print(f"Parsed log: {parsed_size:,} bytes ({parsed_size/(1024*1024):.2f} MB)")
print()

# Count parsed records
with open(parsed_file, 'r') as f:
    parsed_count = sum(1 for line in f if line.startswith('=== Record '))
print(f"Records in parsed file: {parsed_count}")
print()

# Count records in original (streaming to handle large files)
frames = 0
total_records = 0
total_capacity = 0
with open(log_file, 'rb') as f:
    while True:
        header = f.read(8)
        if len(header) < 8:
            break
        capacity = struct.unpack('<I', header[0:4])[0]
        valid_data = struct.unpack('<I', header[4:8])[0]
        frames += 1
        total_capacity += capacity
        
        if valid_data > 0 and capacity >= 8:
            data_size = capacity - 8
            data = f.read(data_size)
            if len(data) < data_size:
                print(f"  [WARNING] Frame {frames}: Partial read")
                break
            
            # Count records
            offset = 0
            while offset < valid_data:
                if offset + 4 > valid_data:
                    break
                rec_len = struct.unpack('<I', data[offset:offset+4])[0]
                offset += 4
                if rec_len == 0:
                    while offset < valid_data and offset % 4 != 0:
                        offset += 1
                    continue
                if offset + rec_len > valid_data:
                    break
                total_records += 1
                offset += rec_len
                while offset < valid_data and offset % 4 != 0:
                    offset += 1
        else:
            if capacity > 8:
                f.read(capacity - 8)
            elif capacity > 0:
                f.read(capacity)
        
        if frames <= 5:
            print(f"Frame {frames}: capacity={capacity:,}, valid_data={valid_data:,}, records={total_records - sum(f.get('records', 0) for f in [])}")

print()
print(f"Total frames: {frames}")
print(f"Total records in log: {total_records}")
print(f"Total capacity: {total_capacity:,} bytes ({total_capacity/(1024*1024):.2f} MB)")
print()
print(f"Record match: {'✓ YES' if total_records == parsed_count else f'✗ NO (log has {total_records}, parsed has {parsed_count}, diff: {abs(total_records - parsed_count)})'}")
print(f"Size match: {'✓ YES' if abs(total_capacity - log_size) < 1000 else f'✗ NO (expected {total_capacity:,}, got {log_size:,}, diff: {abs(total_capacity - log_size):,} bytes)'}")




