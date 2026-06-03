#!/usr/bin/env python3
"""
Round-trip verification:
1. Parse original file and extract all records (preserving raw protobuf bytes)
2. Reconstruct frames with same structure
3. Compare with original
"""

import struct
import sys
from pathlib import Path

def deframe_and_extract_records(log_path: Path):
    """Extract all records from log file, preserving raw bytes"""
    records = []
    frames_info = []
    
    HEADER_SIZE = 8
    LENGTH_PREFIX_SIZE = 4
    
    with open(log_path, 'rb') as f:
        frame_count = 0
        while True:
            frame_start = f.tell()
            header = f.read(HEADER_SIZE)
            if len(header) < HEADER_SIZE:
                break
            
            capacity = int.from_bytes(header[0:4], byteorder='little')
            valid_data_bytes = int.from_bytes(header[4:8], byteorder='little')
            frame_count += 1
            
            if valid_data_bytes == 0:
                # Empty frame
                if capacity > HEADER_SIZE:
                    f.read(capacity - HEADER_SIZE)
                elif capacity > 0:
                    f.read(capacity)
                frames_info.append({
                    'frame': frame_count,
                    'capacity': capacity,
                    'valid_data': 0,
                    'records': []
                })
                continue
            
            if capacity < HEADER_SIZE:
                print(f"[WARNING] Invalid capacity {capacity} at frame {frame_count}")
                continue
            
            # Read data section
            data_size = capacity - HEADER_SIZE
            data_block = f.read(data_size)
            
            if len(data_block) < data_size:
                print(f"[WARNING] Partial read at frame {frame_count}")
                break
            
            # Extract records
            frame_records = []
            offset = 0
            while offset < valid_data_bytes:
                if offset + LENGTH_PREFIX_SIZE > valid_data_bytes:
                    break
                
                record_length = int.from_bytes(
                    data_block[offset:offset+LENGTH_PREFIX_SIZE], 
                    byteorder='little'
                )
                offset += LENGTH_PREFIX_SIZE
                
                if record_length == 0:
                    # Skip padding
                    while offset < valid_data_bytes and offset % LENGTH_PREFIX_SIZE != 0:
                        offset += 1
                    continue
                
                if offset + record_length > valid_data_bytes:
                    break
                
                # Extract record (raw protobuf bytes)
                record = data_block[offset:offset+record_length]
                frame_records.append(record)
                records.append(record)
                offset += record_length
                
                # Skip padding
                while offset < valid_data_bytes and offset % LENGTH_PREFIX_SIZE != 0:
                    offset += 1
            
            frames_info.append({
                'frame': frame_count,
                'capacity': capacity,
                'valid_data': valid_data_bytes,
                'records': frame_records
            })
            
            if frame_count <= 5:
                print(f"Frame {frame_count}: {len(frame_records)} records, capacity={capacity:,}, valid_data={valid_data_bytes:,}")
    
    return records, frames_info


def reconstruct_file(records: List[bytes], frames_info: List[dict], output_path: Path):
    """Reconstruct binary log file from records using original frame structure"""
    HEADER_SIZE = 8
    LENGTH_PREFIX_SIZE = 4
    
    with open(output_path, 'wb') as f:
        record_idx = 0
        
        for frame_info in frames_info:
            capacity = frame_info['capacity']
            original_records = frame_info['records']
            
            # Build data section
            data_section = bytearray()
            
            for record in original_records:
                record_len = len(record)
                
                # Align to 4-byte boundary
                current_len = len(data_section)
                padding = (4 - (current_len % 4)) % 4
                data_section.extend(b'\x00' * padding)
                
                # Add length prefix
                data_section.extend(struct.pack('<I', record_len))
                
                # Add record
                data_section.extend(record)
            
            # Pad to capacity - HEADER_SIZE
            data_size = capacity - HEADER_SIZE
            while len(data_section) < data_size:
                data_section.append(0)
            
            # Truncate if needed
            if len(data_section) > data_size:
                data_section = data_section[:data_size]
            
            # Calculate valid_data_bytes (actual data, aligned to 4-byte boundary)
            valid_data_bytes = len(data_section)
            # Remove trailing zeros but keep 4-byte alignment
            while valid_data_bytes > 0 and data_section[valid_data_bytes - 1] == 0:
                valid_data_bytes -= 1
            valid_data_bytes = ((valid_data_bytes + 3) // 4) * 4
            
            # Write header
            f.write(struct.pack('<I', capacity))
            f.write(struct.pack('<I', valid_data_bytes))
            
            # Write data
            f.write(data_section)
            
            record_idx += len(original_records)
    
    return record_idx


def compare_files_binary(file1: Path, file2: Path):
    """Compare two binary files"""
    size1 = file1.stat().st_size
    size2 = file2.stat().st_size
    
    print(f"\nOriginal file: {size1:,} bytes ({size1/(1024*1024):.2f} MB)")
    print(f"Reconstructed: {size2:,} bytes ({size2/(1024*1024):.2f} MB)")
    
    if size1 != size2:
        print(f"✗ Size mismatch: {abs(size1 - size2):,} bytes difference")
        return False
    
    # Compare byte by byte
    print("Comparing files byte-by-byte...")
    differences = 0
    max_diffs = 20
    
    with open(file1, 'rb') as f1, open(file2, 'rb') as f2:
        pos = 0
        while pos < size1:
            b1 = f1.read(1)
            b2 = f2.read(1)
            if not b1 or not b2:
                break
            if b1 != b2:
                differences += 1
                if differences <= max_diffs:
                    print(f"  Difference at position {pos:,}: 0x{b1.hex()} vs 0x{b2.hex()}")
            pos += 1
            
            # Progress indicator
            if pos % (10 * 1024 * 1024) == 0:
                print(f"  Compared {pos/(1024*1024):.1f} MB...")
    
    if differences == 0:
        print("✓ Files are identical!")
        return True
    else:
        print(f"✗ Found {differences} differences (showing first {max_diffs})")
        return False


def main():
    original_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log')
    reconstructed_file = Path('reconstructed_verification.log')
    
    if not original_file.exists():
        print(f"Error: {original_file} not found")
        sys.exit(1)
    
    print("=" * 80)
    print("ROUND-TRIP VERIFICATION")
    print("=" * 80)
    print()
    
    print("Step 1: Extracting records from original file...")
    records, frames_info = deframe_and_extract_records(original_file)
    print(f"Extracted {len(records)} records from {len(frames_info)} frames")
    print()
    
    print("Step 2: Reconstructing binary log file...")
    records_written = reconstruct_file(records, frames_info, reconstructed_file)
    print(f"Reconstructed {records_written} records into {len(frames_info)} frames")
    print()
    
    print("Step 3: Comparing files...")
    match = compare_files_binary(original_file, reconstructed_file)
    print()
    
    if match:
        print("=" * 80)
        print("✓✓✓ VERIFICATION PASSED - Files are identical! ✓✓✓")
        print("=" * 80)
        # Cleanup
        reconstructed_file.unlink()
        print("Reconstructed file deleted (verification passed)")
    else:
        print("=" * 80)
        print("✗✗✗ VERIFICATION FAILED - Files differ ✗✗✗")
        print("=" * 80)
        print(f"\nReconstructed file saved to: {reconstructed_file}")
        print("You can inspect the differences manually.")


if __name__ == '__main__':
    main()




