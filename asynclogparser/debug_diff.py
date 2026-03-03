#!/usr/bin/env python3
"""Debug differences between original and reconstructed files"""
import struct
from pathlib import Path

original_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log')
reconstructed_file = Path('Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.reconstructed.log')

print("Comparing first 1KB of files...")
print("=" * 80)

with open(original_file, 'rb') as f1, open(reconstructed_file, 'rb') as f2:
    # Read first 1KB
    orig_data = f1.read(1024)
    recon_data = f2.read(1024)
    
    # Compare byte by byte
    differences = []
    for i in range(min(len(orig_data), len(recon_data))):
        if orig_data[i] != recon_data[i]:
            differences.append((i, orig_data[i], recon_data[i]))
    
    print(f"Found {len(differences)} byte differences in first 1KB")
    print()
    
    if differences:
        print("First 20 differences:")
        for i, (pos, b1, b2) in enumerate(differences[:20]):
            print(f"  Position {pos:4d} (0x{pos:04x}): 0x{b1:02x} vs 0x{b2:02x} ({b1} vs {b2})")
        print()
    
    # Check header
    print("Frame 1 Header Comparison:")
    print("-" * 80)
    orig_header = orig_data[0:8]
    recon_header = recon_data[0:8]
    
    orig_capacity = struct.unpack('<I', orig_header[0:4])[0]
    orig_valid = struct.unpack('<I', orig_header[4:8])[0]
    recon_capacity = struct.unpack('<I', recon_header[0:4])[0]
    recon_valid = struct.unpack('<I', recon_header[4:8])[0]
    
    print(f"Original:   capacity={orig_capacity:,} ({orig_capacity/(1024*1024):.2f}MB), valid_data={orig_valid:,} ({orig_valid/(1024*1024):.2f}MB)")
    print(f"Reconstructed: capacity={recon_capacity:,} ({recon_capacity/(1024*1024):.2f}MB), valid_data={recon_valid:,} ({recon_valid/(1024*1024):.2f}MB)")
    print()
    
    if orig_header != recon_header:
        print("✗ Headers differ!")
        print(f"  Original:   {orig_header.hex()}")
        print(f"  Reconstructed: {recon_header.hex()}")
    else:
        print("✓ Headers match")
    print()
    
    # Check first record
    print("First Record Comparison:")
    print("-" * 80)
    
    # Read length prefix (4 bytes after header)
    orig_len = struct.unpack('<I', orig_data[8:12])[0]
    recon_len = struct.unpack('<I', recon_data[8:12])[0]
    
    print(f"Original record length: {orig_len:,} bytes")
    print(f"Reconstructed record length: {recon_len:,} bytes")
    print()
    
    if orig_len == recon_len:
        # Compare record data
        orig_record = orig_data[12:12+orig_len]
        recon_record = recon_data[12:12+recon_len]
        
        if len(orig_record) == len(recon_record):
            record_diffs = sum(1 for i in range(len(orig_record)) if orig_record[i] != recon_record[i])
            print(f"Record data differences: {record_diffs} bytes")
            
            if record_diffs > 0:
                print("First 10 record data differences:")
                count = 0
                for i in range(len(orig_record)):
                    if orig_record[i] != recon_record[i] and count < 10:
                        print(f"  Position {i}: 0x{orig_record[i]:02x} vs 0x{recon_record[i]:02x}")
                        count += 1
        else:
            print(f"✗ Record length mismatch: {len(orig_record)} vs {len(recon_record)}")
    else:
        print("✗ Record length prefix differs!")




