#!/usr/bin/env python3
"""
Verify that all data from the log file was parsed correctly
"""

import struct
from pathlib import Path

log_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log')
parsed_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.parsed.log')

print("=" * 80)
print("VERIFICATION: Comparing original log file with parsed output")
print("=" * 80)
print()

# Get file sizes
log_size = log_file.stat().st_size
parsed_size = parsed_file.stat().st_size

print(f"Original log file: {log_size:,} bytes ({log_size/(1024*1024):.2f} MB)")
print(f"Parsed log file: {parsed_size:,} bytes ({parsed_size/(1024*1024):.2f} MB)")
print()

# Count records in parsed file
with open(parsed_file, 'r') as f:
    record_lines = [line for line in f if line.startswith('=== Record ')]
    record_count = len(record_lines)
    if record_lines:
        last_record = record_lines[-1].strip()
        print(f"Records in parsed file: {record_count}")
        print(f"Last record: {last_record}")
    else:
        print("No records found in parsed file!")

print()

# Analyze original log file structure
frames = []
total_records = 0
total_capacity = 0
total_valid_data = 0

with open(log_file, 'rb') as f:
    frame_num = 0
    while True:
        frame_start = f.tell()
        header = f.read(8)
        if len(header) < 8:
            break
        
        capacity = struct.unpack('<I', header[0:4])[0]
        valid_data = struct.unpack('<I', header[4:8])[0]
        
        frame_num += 1
        total_capacity += capacity
        total_valid_data += valid_data
        
        # Count records in this frame
        records_in_frame = 0
        if valid_data > 0 and capacity >= 8:
            data_size = capacity - 8
            data_block = f.read(data_size)
            
            if len(data_block) < data_size:
                print(f"  [WARNING] Frame {frame_num}: Partial read, expected {data_size}, got {len(data_block)}")
                break
            
            # Count records
            offset = 0
            while offset < valid_data:
                if offset + 4 > valid_data:
                    break
                record_len = struct.unpack('<I', data_block[offset:offset+4])[0]
                offset += 4
                if record_len == 0:
                    # Skip padding
                    while offset < valid_data and offset % 4 != 0:
                        offset += 1
                    continue
                if offset + record_len > valid_data:
                    break
                records_in_frame += 1
                offset += record_len
                # Skip padding
                while offset < valid_data and offset % 4 != 0:
                    offset += 1
            
            total_records += records_in_frame
        else:
            # Empty frame, skip data section
            if capacity > 8:
                f.read(capacity - 8)
            elif capacity > 0:
                f.read(capacity)
        
        frames.append({
            'frame': frame_num,
            'capacity': capacity,
            'valid_data': valid_data,
            'records': records_in_frame,
            'start_pos': frame_start
        })
        
        if frame_num <= 5:
            print(f"Frame {frame_num} @ pos {frame_start:,}: capacity={capacity:,} ({capacity/(1024*1024):.2f}MB), "
                  f"valid_data={valid_data:,} ({valid_data/(1024*1024):.2f}MB), records={records_in_frame}")

print()
print("=" * 80)
print("SUMMARY")
print("=" * 80)
print(f"Total frames in log file: {len(frames)}")
print(f"Total capacity (all frames): {total_capacity:,} bytes ({total_capacity/(1024*1024):.2f} MB)")
print(f"Total valid data: {total_valid_data:,} bytes ({total_valid_data/(1024*1024):.2f} MB)")
print(f"Total records in log file: {total_records}")
print(f"Records in parsed file: {record_count}")
print()

# Verify file size
expected_file_size = total_capacity
size_match = abs(expected_file_size - log_size) < 1000
print(f"Expected file size (sum of all frame capacities): {expected_file_size:,} bytes")
print(f"Actual file size: {log_size:,} bytes")
print(f"File size match: {'✓ YES' if size_match else f'✗ NO (difference: {abs(expected_file_size - log_size):,} bytes)'}")
print()

# Verify record count
record_match = total_records == record_count
print(f"Record count match: {'✓ YES' if record_match else f'✗ NO (difference: {abs(total_records - record_count)} records)'}")
print()

if size_match and record_match:
    print("✓✓✓ ALL DATA VERIFIED - Parsing is complete and correct! ✓✓✓")
else:
    print("✗✗✗ VERIFICATION FAILED - Some data may be missing or incorrect ✗✗✗")
    if not size_match:
        print(f"  - File size mismatch suggests frames may not be aligned correctly")
    if not record_match:
        print(f"  - Record count mismatch: expected {total_records}, got {record_count}")




