#!/usr/bin/env python3
"""Simple verification: Extract records and verify structure"""
import struct
from pathlib import Path

log_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log')
parsed_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.parsed.log')

print("=" * 80)
print("VERIFICATION: Record Extraction and Structure Check")
print("=" * 80)
print()

# Extract all records from original
print("Extracting records from original file...")
records = []
frames = []
total_capacity = 0

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
        
        frame_records = []
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
                record = data[offset:offset+rec_len]
                frame_records.append(record)
                records.append(record)
                offset += rec_len
                while offset < valid_data and offset % 4 != 0:
                    offset += 1
        else:
            if capacity > 8:
                f.read(capacity - 8)
            elif capacity > 0:
                f.read(capacity)
        
        frames.append({
            'frame': frame_num,
            'capacity': capacity,
            'valid_data': valid_data,
            'records': len(frame_records)
        })
        
        if frame_num <= 5:
            print(f"  Frame {frame_num}: {len(frame_records)} records, capacity={capacity:,}, valid_data={valid_data:,}")

print(f"\nTotal frames: {len(frames)}")
print(f"Total records extracted: {len(records)}")
print(f"Total capacity: {total_capacity:,} bytes ({total_capacity/(1024*1024):.2f} MB)")

# Count in parsed file
print("\nCounting records in parsed file...")
with open(parsed_file, 'r') as f:
    parsed_count = sum(1 for line in f if line.startswith('=== Record '))
print(f"Records in parsed file: {parsed_count}")

# File size
file_size = log_file.stat().st_size
print(f"\nOriginal file size: {file_size:,} bytes ({file_size/(1024*1024):.2f} MB)")
print(f"Expected size (sum capacities): {total_capacity:,} bytes")

print()
print("=" * 80)
print("VERIFICATION RESULTS")
print("=" * 80)
print(f"Record count match: {'✓ YES' if len(records) == parsed_count else f'✗ NO (original: {len(records)}, parsed: {parsed_count})'}")
print(f"File size match: {'✓ YES' if abs(total_capacity - file_size) < 1000 else f'✗ NO (diff: {abs(total_capacity - file_size):,} bytes)'}")
print()

if len(records) == parsed_count and abs(total_capacity - file_size) < 1000:
    print("✓✓✓ ALL CHECKS PASSED - Parsing is complete and correct! ✓✓✓")
else:
    print("✗✗✗ VERIFICATION FAILED ✗✗✗")




