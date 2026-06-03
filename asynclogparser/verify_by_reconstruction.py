#!/usr/bin/env python3
"""
Verification by reconstruction:
1. Parse original file and extract all records (as raw protobuf bytes)
2. Reconstruct frames with the same structure
3. Compare reconstructed file with original
"""

import struct
from pathlib import Path
from typing import List, Tuple

def extract_all_records(original_file: Path) -> Tuple[List[bytes], List[dict]]:
    """
    Extract all records from original file, preserving raw protobuf bytes.
    Returns: (list of record bytes, list of frame metadata)
    """
    records = []
    frames_info = []
    
    with open(original_file, 'rb') as f:
        frame_num = 0
        while True:
            frame_start = f.tell()
            header = f.read(8)
            if len(header) < 8:
                break
            
            capacity = struct.unpack('<I', header[0:4])[0]
            valid_data = struct.unpack('<I', header[4:8])[0]
            frame_num += 1
            
            frame_records = []
            if valid_data > 0 and capacity >= 8:
                data_size = capacity - 8
                data_block = f.read(data_size)
                
                if len(data_block) < data_size:
                    print(f"  [WARNING] Frame {frame_num}: Partial read")
                    break
                
                # Extract records
                offset = 0
                while offset < valid_data:
                    if offset + 4 > valid_data:
                        break
                    record_len = struct.unpack('<I', data_block[offset:offset+4])[0]
                    offset += 4
                    if record_len == 0:
                        while offset < valid_data and offset % 4 != 0:
                            offset += 1
                        continue
                    if offset + record_len > valid_data:
                        break
                    
                    # Extract the record (raw protobuf bytes)
                    record = data_block[offset:offset+record_len]
                    frame_records.append(record)
                    records.append(record)
                    offset += record_len
                    while offset < valid_data and offset % 4 != 0:
                        offset += 1
            else:
                # Empty frame
                if capacity > 8:
                    f.read(capacity - 8)
                elif capacity > 0:
                    f.read(capacity)
            
            frames_info.append({
                'frame': frame_num,
                'start_pos': frame_start,
                'capacity': capacity,
                'valid_data': valid_data,
                'records': frame_records,
                'num_records': len(frame_records)
            })
            
            if frame_num <= 5:
                print(f"Frame {frame_num}: {len(frame_records)} records, capacity={capacity:,}, valid_data={valid_data:,}")
    
    return records, frames_info


def reconstruct_frames(records: List[bytes], frames_info: List[dict], output_file: Path):
    """
    Reconstruct binary log file from extracted records using original frame structure.
    """
    header_size = 8
    length_prefix_size = 4
    
    with open(output_file, 'wb') as f:
        record_idx = 0
        
        for frame_info in frames_info:
            # Reconstruct frame
            capacity = frame_info['capacity']
            num_records_in_frame = frame_info['num_records']
            
            # Calculate how much data we can fit
            # Frame structure: [8-byte header][data section]
            # Data section: [length:4][data:N] [length:4][data:M] ... [padding]
            data_section_size = capacity - header_size
            
            # Build data section
            data_section = bytearray()
            records_written = 0
            
            for i in range(num_records_in_frame):
                if record_idx >= len(records):
                    break
                
                record = records[record_idx]
                record_len = len(record)
                
                # Check if record fits
                needed = length_prefix_size + record_len
                # Align to 4-byte boundary
                current_offset = len(data_section)
                padding_needed = (4 - (current_offset % 4)) % 4
                
                if current_offset + padding_needed + needed > data_section_size:
                    # Record doesn't fit, stop
                    break
                
                # Add padding if needed
                data_section.extend(b'\x00' * padding_needed)
                
                # Add length prefix
                data_section.extend(struct.pack('<I', record_len))
                
                # Add record data
                data_section.extend(record)
                
                records_written += 1
                record_idx += 1
            
            # Pad data section to capacity - 8
            while len(data_section) < data_section_size:
                data_section.append(0)
            
            # Truncate if too large (shouldn't happen)
            if len(data_section) > data_section_size:
                data_section = data_section[:data_section_size]
            
            # Calculate valid_data_bytes (actual data written, excluding padding)
            valid_data_bytes = len(data_section)
            # Find last non-zero byte or align to 4-byte boundary
            while valid_data_bytes > 0 and data_section[valid_data_bytes - 1] == 0:
                valid_data_bytes -= 1
            # Align to 4-byte boundary
            valid_data_bytes = ((valid_data_bytes + 3) // 4) * 4
            
            # Write header
            f.write(struct.pack('<I', capacity))
            f.write(struct.pack('<I', valid_data_bytes))
            
            # Write data section
            f.write(data_section)
            
            if frame_info['frame'] <= 5:
                print(f"Reconstructed Frame {frame_info['frame']}: {records_written}/{num_records_in_frame} records, "
                      f"capacity={capacity:,}, valid_data={valid_data_bytes:,}")
    
    return record_idx


def compare_files(file1: Path, file2: Path, max_diff_bytes: int = 100):
    """
    Compare two binary files and report differences.
    """
    size1 = file1.stat().st_size
    size2 = file2.stat().st_size
    
    print(f"\nFile 1 ({file1.name}): {size1:,} bytes")
    print(f"File 2 ({file2.name}): {size2:,} bytes")
    
    if size1 != size2:
        print(f"✗ Size mismatch: difference of {abs(size1 - size2):,} bytes")
        return False
    
    # Compare byte by byte (for first few differences)
    differences = []
    with open(file1, 'rb') as f1, open(file2, 'rb') as f2:
        pos = 0
        while pos < size1 and len(differences) < max_diff_bytes:
            b1 = f1.read(1)
            b2 = f2.read(1)
            if b1 != b2:
                differences.append((pos, b1, b2))
            pos += 1
    
    if differences:
        print(f"✗ Found {len(differences)} byte differences (showing first {min(10, len(differences))}):")
        for pos, b1, b2 in differences[:10]:
            print(f"  Position {pos:,}: 0x{b1.hex()} vs 0x{b2.hex()}")
        return False
    else:
        print("✓ Files are identical!")
        return True


def main():
    original_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log')
    reconstructed_file = Path('reconstructed.log')
    
    print("=" * 80)
    print("VERIFICATION BY RECONSTRUCTION")
    print("=" * 80)
    print()
    print("Step 1: Extracting all records from original file...")
    records, frames_info = extract_all_records(original_file)
    print(f"Extracted {len(records)} records from {len(frames_info)} frames")
    print()
    
    print("Step 2: Reconstructing binary log file...")
    records_written = reconstruct_frames(records, frames_info, reconstructed_file)
    print(f"Reconstructed {records_written} records into {len(frames_info)} frames")
    print()
    
    print("Step 3: Comparing original and reconstructed files...")
    match = compare_files(original_file, reconstructed_file)
    print()
    
    if match:
        print("=" * 80)
        print("✓✓✓ VERIFICATION PASSED - Files match exactly! ✓✓✓")
        print("=" * 80)
    else:
        print("=" * 80)
        print("✗✗✗ VERIFICATION FAILED - Files differ ✗✗✗")
        print("=" * 80)
        print("\nNote: Some differences may be expected due to:")
        print("  - Padding bytes (zeros) may differ")
        print("  - Frame capacity may be pre-allocated")
        print("  - Exact byte alignment may vary")
    
    # Cleanup
    if reconstructed_file.exists():
        print(f"\nReconstructed file saved to: {reconstructed_file}")
        print("You can manually compare it with the original if needed.")


if __name__ == '__main__':
    main()




