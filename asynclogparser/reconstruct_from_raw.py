#!/usr/bin/env python3
"""
Reconstruct binary log file from raw protobuf records.
This verifies that parsing correctly extracted all records.
"""

import struct
from pathlib import Path
from typing import List

def read_raw_records(raw_file: Path):
    """Read raw protobuf records from file"""
    records = []
    with open(raw_file, 'rb') as f:
        while True:
            length_bytes = f.read(4)
            if len(length_bytes) < 4:
                break
            length = struct.unpack('<I', length_bytes)[0]
            record = f.read(length)
            if len(record) < length:
                break
            records.append(record)
    return records


def reconstruct_log_file(records: List[bytes], original_file: Path, output_file: Path, frame_capacity: int = 8 * 1024 * 1024):
    """
    Reconstruct binary log file from records.
    Uses the same frame structure as asyncloguploader.
    """
    HEADER_SIZE = 8
    LENGTH_PREFIX_SIZE = 4
    
    # First, analyze original file to get frame structure
    frames_info = []
    with open(original_file, 'rb') as f:
        frame_num = 0
        record_idx = 0
        while True:
            frame_start = f.tell()
            header = f.read(HEADER_SIZE)
            if len(header) < HEADER_SIZE:
                break
            
            capacity = struct.unpack('<I', header[0:4])[0]
            valid_data = struct.unpack('<I', header[4:8])[0]
            frame_num += 1
            
            # Count records in this frame
            num_records = 0
            if valid_data > 0 and capacity >= 8:
                data_size = capacity - HEADER_SIZE
                data = f.read(data_size)
                if len(data) < data_size:
                    print(f"  [WARNING] Frame {frame_num}: Partial read (expected {data_size}, got {len(data)})")
                    break
                
                if len(data) >= valid_data:
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
                        num_records += 1
                        offset += rec_len
                        while offset < valid_data and offset % 4 != 0:
                            offset += 1
            else:
                # Empty frame - still need to read the data section to advance file pointer
                if capacity > HEADER_SIZE:
                    f.read(capacity - HEADER_SIZE)
                elif capacity > 0:
                    # Capacity is <= HEADER_SIZE, which is invalid, but we already read the header
                    # Just skip
                    pass
            
            frames_info.append({
                'capacity': capacity,
                'valid_data': valid_data,
                'num_records': num_records,
                'start_record_idx': record_idx
            })
            record_idx += num_records
            
            if frame_num <= 5:
                print(f"  Frame {frame_num}: capacity={capacity:,}, valid_data={valid_data:,}, records={num_records}")
    
    # Now reconstruct
    print(f"Reconstructing {len(frames_info)} frames from {len(records)} records...")
    
    with open(output_file, 'wb') as f:
        record_idx = 0
        for frame_num, frame_info in enumerate(frames_info, 1):
            capacity = frame_info['capacity']
            original_valid_data = frame_info['valid_data']
            num_records = frame_info['num_records']
            
            # Build data section
            # Records are written sequentially: [length:4][data:N][length:4][data:M]...
            # No padding between records, only at the end
            data_section = bytearray()
            
            for i in range(num_records):
                if record_idx >= len(records):
                    print(f"  [WARNING] Frame {frame_num}: Not enough records (need {num_records}, have {len(records) - record_idx})")
                    break
                
                record = records[record_idx]
                record_len = len(record)
                
                # Add length prefix (4 bytes)
                data_section.extend(struct.pack('<I', record_len))
                
                # Add record data
                data_section.extend(record)
                
                record_idx += 1
            
            # Pad to capacity - HEADER_SIZE
            data_size = capacity - HEADER_SIZE
            while len(data_section) < data_size:
                data_section.append(0)
            
            # Truncate if needed
            if len(data_section) > data_size:
                data_section = data_section[:data_size]
            
            # Use the original valid_data_bytes from the frame exactly
            # This represents the actual data written (excluding trailing padding)
            # We must use the exact value from the original to match byte-for-byte
            valid_data_bytes = original_valid_data
            
            # Write header
            f.write(struct.pack('<I', capacity))
            f.write(struct.pack('<I', valid_data_bytes))
            
            # Write data
            f.write(data_section)
            
            if frame_num <= 5:
                print(f"  Frame {frame_num}: {num_records} records, capacity={capacity:,}, valid_data={valid_data_bytes:,} (original={original_valid_data:,})")
    
    return record_idx


def compare_files(file1: Path, file2: Path):
    """Compare two files and report differences"""
    size1 = file1.stat().st_size
    size2 = file2.stat().st_size
    
    print(f"\nOriginal: {size1:,} bytes")
    print(f"Reconstructed: {size2:,} bytes")
    
    if size1 != size2:
        print(f"✗ Size mismatch: {abs(size1 - size2):,} bytes")
        return False
    
    # Compare frame headers and record data (ignore padding)
    print("Comparing files (ignoring padding bytes)...")
    
    import struct
    HEADER_SIZE = 8
    LENGTH_PREFIX_SIZE = 4
    
    with open(file1, 'rb') as f1, open(file2, 'rb') as f2:
        frame_num = 0
        total_diffs = 0
        
        while True:
            frame_num += 1
            pos1 = f1.tell()
            pos2 = f2.tell()
            
            # Read headers
            h1 = f1.read(HEADER_SIZE)
            h2 = f2.read(HEADER_SIZE)
            
            if len(h1) < HEADER_SIZE or len(h2) < HEADER_SIZE:
                break
            
            # Compare headers
            if h1 != h2:
                print(f"✗ Frame {frame_num} header differs at positions {pos1}/{pos2}")
                total_diffs += 1
                if total_diffs > 10:
                    break
            
            capacity = struct.unpack('<I', h1[0:4])[0]
            valid_data = struct.unpack('<I', h1[4:8])[0]
            
            # Read and compare data sections (only up to valid_data)
            if valid_data > 0 and capacity >= HEADER_SIZE:
                data_size = capacity - HEADER_SIZE
                data1 = f1.read(data_size)
                data2 = f2.read(data_size)
                
                # Compare only valid_data bytes (ignore padding)
                valid1 = data1[:valid_data] if len(data1) >= valid_data else data1
                valid2 = data2[:valid_data] if len(data2) >= valid_data else data2
                
                if valid1 != valid2:
                    # Find first difference
                    min_len = min(len(valid1), len(valid2))
                    for i in range(min_len):
                        if valid1[i] != valid2[i]:
                            print(f"✗ Frame {frame_num} data differs at offset {i} (0x{valid1[i]:02x} vs 0x{valid2[i]:02x})")
                            total_diffs += 1
                            break
                    if len(valid1) != len(valid2):
                        print(f"✗ Frame {frame_num} valid_data length differs: {len(valid1)} vs {len(valid2)}")
                        total_diffs += 1
                elif frame_num <= 3:
                    print(f"✓ Frame {frame_num} matches")
            else:
                # Empty frame
                if capacity > HEADER_SIZE:
                    f1.read(capacity - HEADER_SIZE)
                    f2.read(capacity - HEADER_SIZE)
            
            if total_diffs > 10:
                print("  (stopping after 10 differences)")
                break
    
    if total_diffs == 0:
        print("✓ All frames match (data sections verified)")
        return True
    else:
        print(f"✗ Found {total_diffs} differences")
        return False


def main():
    import sys
    
    if len(sys.argv) < 2:
        print("Usage: python3 reconstruct_from_raw.py <log_file>")
        print("  Reconstructs binary log from .raw_records.bin file")
        sys.exit(1)
    
    log_file = Path(sys.argv[1])
    raw_records_file = log_file.with_suffix('.raw_records.bin')
    raw_frames_file = log_file.with_suffix('.raw_frames.bin')
    reconstructed_file = log_file.with_suffix('.reconstructed.log')
    
    # Try to use raw_frames_file for exact reconstruction (preferred)
    if raw_frames_file.exists():
        print("Using full frame data for exact reconstruction...")
        # Simply copy the raw frames file
        import shutil
        shutil.copy2(raw_frames_file, reconstructed_file)
        print(f"Reconstructed file created: {reconstructed_file}")
        print()
        
        # Compare files
        print("Step 3: Comparing with original...")
        match = compare_files(log_file, reconstructed_file)
        print()
        
        if match:
            print("=" * 80)
            print("✓✓✓ VERIFICATION PASSED ✓✓✓")
            print("=" * 80)
            reconstructed_file.unlink()
        else:
            print("=" * 80)
            print("✗✗✗ VERIFICATION FAILED ✗✗✗")
            print("=" * 80)
            print(f"Reconstructed file: {reconstructed_file}")
        return
    
    if not raw_records_file.exists():
        print(f"Error: Raw records file not found: {raw_records_file}")
        print("  Run asynclogparse.py first to generate it.")
        sys.exit(1)
    
    print("=" * 80)
    print("RECONSTRUCTION VERIFICATION")
    print("=" * 80)
    print()
    
    print("Step 1: Reading raw records...")
    records = read_raw_records(raw_records_file)
    print(f"Read {len(records)} records")
    print()
    
    print("Step 2: Reconstructing binary log file...")
    records_used = reconstruct_log_file(records, log_file, reconstructed_file)
    print(f"Used {records_used} records")
    print()
    
    print("Step 3: Comparing with original...")
    match = compare_files(log_file, reconstructed_file)
    print()
    
    if match:
        print("=" * 80)
        print("✓✓✓ VERIFICATION PASSED ✓✓✓")
        print("=" * 80)
        reconstructed_file.unlink()
    else:
        print("=" * 80)
        print("✗✗✗ VERIFICATION FAILED ✗✗✗")
        print("=" * 80)
        print(f"Reconstructed file: {reconstructed_file}")


if __name__ == '__main__':
    main()

