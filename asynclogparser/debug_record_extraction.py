#!/usr/bin/env python3
"""Debug record extraction to verify we're getting exact bytes"""
import struct
from pathlib import Path

log_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log')
raw_records_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.raw_records.bin')

HEADER_SIZE = 8
LENGTH_PREFIX_SIZE = 4

print("Extracting records from original file...")
original_records = []
with open(log_file, 'rb') as f:
    frame_num = 0
    while frame_num < 3:  # Check first 3 frames
        header = f.read(HEADER_SIZE)
        if len(header) < HEADER_SIZE:
            break
        
        capacity = struct.unpack('<I', header[0:4])[0]
        valid_data = struct.unpack('<I', header[4:8])[0]
        frame_num += 1
        
        if valid_data > 0 and capacity >= HEADER_SIZE:
            data_size = capacity - HEADER_SIZE
            data_block = f.read(data_size)
            
            # Extract first record
            if len(data_block) >= LENGTH_PREFIX_SIZE:
                record_len = struct.unpack('<I', data_block[0:4])[0]
                if record_len > 0 and record_len <= len(data_block) - LENGTH_PREFIX_SIZE:
                    record = data_block[LENGTH_PREFIX_SIZE:LENGTH_PREFIX_SIZE+record_len]
                    original_records.append({
                        'frame': frame_num,
                        'record': record,
                        'offset_in_frame': LENGTH_PREFIX_SIZE,
                        'length': record_len
                    })
                    
                    # Check bytes at offset 2434 (relative to start of data block)
                    if len(data_block) > 2434:
                        print(f"\nFrame {frame_num}:")
                        print(f"  Record length: {record_len:,} bytes")
                        print(f"  Data block length: {len(data_block):,} bytes")
                        print(f"  Byte at offset 2434 in data_block: 0x{data_block[2434]:02x}")
                        print(f"  Byte at offset 2434 in record: ", end="")
                        record_offset = 2434 - LENGTH_PREFIX_SIZE
                        if 0 <= record_offset < len(record):
                            print(f"0x{record[record_offset]:02x}")
                        else:
                            print("(outside record)")
                        
                        # Show context around offset 2434
                        start = max(0, 2434 - 10)
                        end = min(len(data_block), 2434 + 10)
                        print(f"  Context in data_block [{start}:{end}]: {data_block[start:end].hex()}")
                        if record_offset >= 0 and record_offset < len(record):
                            rec_start = max(0, record_offset - 10)
                            rec_end = min(len(record), record_offset + 10)
                            print(f"  Context in record [{rec_start}:{rec_end}]: {record[rec_start:rec_end].hex()}")

print("\n\nReading records from raw_records.bin...")
reconstructed_records = []
with open(raw_records_file, 'rb') as f:
    record_num = 0
    while record_num < 3:
        length_bytes = f.read(4)
        if len(length_bytes) < 4:
            break
        length = struct.unpack('<I', length_bytes)[0]
        record = f.read(length)
        if len(record) < length:
            break
        reconstructed_records.append({
            'record_num': record_num + 1,
            'record': record,
            'length': length
        })
        record_num += 1
        
        if record_num <= 3:
            print(f"\nRecord {record_num}:")
            print(f"  Length: {length:,} bytes")
            if len(record) > 2430:
                print(f"  Byte at offset 2430 in record: 0x{record[2430]:02x}")
                start = max(0, 2430 - 10)
                end = min(len(record), 2430 + 10)
                print(f"  Context [{start}:{end}]: {record[start:end].hex()}")

print("\n\nComparing original vs extracted records...")
for i in range(min(len(original_records), len(reconstructed_records))):
    orig = original_records[i]
    recon = reconstructed_records[i]
    
    print(f"\nFrame {orig['frame']} / Record {recon['record_num']}:")
    print(f"  Original length: {orig['length']:,}")
    print(f"  Extracted length: {recon['length']:,}")
    
    if orig['length'] == recon['length']:
        if orig['record'] == recon['record']:
            print("  ✓ Records match exactly")
        else:
            # Find first difference
            for j in range(min(len(orig['record']), len(recon['record']))):
                if orig['record'][j] != recon['record'][j]:
                    print(f"  ✗ First difference at offset {j}: 0x{orig['record'][j]:02x} vs 0x{recon['record'][j]:02x}")
                    break
    else:
        print("  ✗ Length mismatch!")




