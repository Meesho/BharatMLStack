#!/usr/bin/env python3
"""Check what's at offset 2434 in the original file"""
import struct
from pathlib import Path

log_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log')

HEADER_SIZE = 8
LENGTH_PREFIX_SIZE = 4

with open(log_file, 'rb') as f:
    # Read first frame
    header = f.read(HEADER_SIZE)
    capacity = struct.unpack('<I', header[0:4])[0]
    valid_data = struct.unpack('<I', header[4:8])[0]
    
    print(f"Frame 1:")
    print(f"  Capacity: {capacity:,} bytes")
    print(f"  Valid data: {valid_data:,} bytes")
    print()
    
    # Read data block
    data_size = capacity - HEADER_SIZE
    data_block = f.read(data_size)
    
    print(f"Data block size: {len(data_block):,} bytes")
    print()
    
    # Check offset 2434
    if len(data_block) > 2434:
        print(f"Byte at offset 2434: 0x{data_block[2434]:02x} ({data_block[2434]})")
        print(f"Context [2424:2444]: {data_block[2424:2444].hex()}")
        print()
    
    # Extract first record
    if len(data_block) >= LENGTH_PREFIX_SIZE:
        record_len = struct.unpack('<I', data_block[0:4])[0]
        print(f"First record length prefix: {record_len:,} bytes")
        print()
        
        if record_len > 0 and record_len <= len(data_block) - LENGTH_PREFIX_SIZE:
            record_start = LENGTH_PREFIX_SIZE
            record_end = record_start + record_len
            record = data_block[record_start:record_end]
            
            print(f"Record extracted: {len(record):,} bytes")
            print(f"Record range in data_block: [{record_start}:{record_end}]")
            print()
            
            # Check offset 2434 relative to record
            offset_in_record = 2434 - LENGTH_PREFIX_SIZE
            print(f"Offset 2434 in data_block corresponds to offset {offset_in_record} in record")
            
            if 0 <= offset_in_record < len(record):
                print(f"  Byte in record at this offset: 0x{record[offset_in_record]:02x}")
            else:
                print(f"  This offset is outside the record (record length: {len(record)})")
                print(f"  But byte in data_block at 2434: 0x{data_block[2434]:02x}")
                print()
                print("  This suggests there's data AFTER the record that we're not capturing!")
                
                # Check what's after the record
                if record_end < len(data_block):
                    print(f"  Data after record [{record_end}:{min(record_end+100, len(data_block))}]:")
                    print(f"    {data_block[record_end:min(record_end+100, len(data_block))].hex()}")
                    
                    # Check if there's another record
                    if record_end + LENGTH_PREFIX_SIZE <= len(data_block):
                        next_len = struct.unpack('<I', data_block[record_end:record_end+LENGTH_PREFIX_SIZE])[0]
                        print(f"  Next length prefix (if any): {next_len:,} bytes")
                        if next_len == 0:
                            print("  (This is padding)")




