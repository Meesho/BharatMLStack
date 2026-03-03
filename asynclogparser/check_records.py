#!/usr/bin/env python3
import struct
import os

log_file = 'Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log'
parsed_file = 'Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.parsed.log'

# File sizes
log_size = os.path.getsize(log_file)
parsed_size = os.path.getsize(parsed_file)
print(f"Original: {log_size:,} bytes ({log_size/(1024*1024):.2f} MB)")
print(f"Parsed: {parsed_size:,} bytes ({parsed_size/(1024*1024):.2f} MB)")
print()

# Count records in parsed file
with open(parsed_file, 'r') as f:
    parsed_records = sum(1 for line in f if line.startswith('=== Record '))
print(f"Records in parsed file: {parsed_records}")
print()

# Count records in original file
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
                break
            
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

print(f"Frames in log file: {frames}")
print(f"Total records in log file: {total_records}")
print(f"Total capacity: {total_capacity:,} bytes ({total_capacity/(1024*1024):.2f} MB)")
print()
print(f"Match: {'✓ YES' if total_records == parsed_records else f'✗ NO (diff: {abs(total_records - parsed_records)})'}")
print(f"File size match: {'✓ YES' if abs(total_capacity - log_size) < 1000 else f'✗ NO (diff: {abs(total_capacity - log_size):,} bytes)'}")




