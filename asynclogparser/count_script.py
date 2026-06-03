#!/usr/bin/env python3
"""
Count entities (records) in asyncloguploader log files.

This script parses the binary log file format and counts the total number
of records/entities stored in the file.

File Format:
- Each frame has an 8-byte header: capacity (4 bytes LE) + validDataBytes (4 bytes LE)
- After header, data section contains records
- Each record: 4-byte length prefix (LE) + record data
"""

import struct
import sys
import json
from pathlib import Path
from typing import Tuple, Optional, List, Dict

# Constants
HEADER_SIZE = 8  # 4 bytes capacity + 4 bytes validDataBytes
LENGTH_PREFIX_SIZE = 4  # 4 bytes for record length


def read_frame_header(f) -> Optional[Tuple[int, int]]:
    """
    Read frame header from file.
    Returns (capacity, valid_data_bytes) or None if EOF.
    """
    header = f.read(HEADER_SIZE)
    if len(header) < HEADER_SIZE:
        return None
    
    capacity = struct.unpack('<I', header[0:4])[0]
    valid_data_bytes = struct.unpack('<I', header[4:8])[0]
    
    return (capacity, valid_data_bytes)


def count_records_in_frame(f, capacity: int, valid_data_bytes: int, collect_sizes: bool = False, analyze_padding: bool = False) -> Tuple[int, List[int], Dict]:
    """
    Count records in a single frame and optionally collect their sizes and analyze padding.
    Returns (record_count, list_of_record_sizes, analysis_dict).
    """
    analysis = {
        'zero_length_records': 0,
        'total_record_bytes': 0,  # Sum of all record data (excluding length prefixes)
        'total_length_prefix_bytes': 0,  # Sum of all 4-byte length prefixes
        'total_consumed_bytes': 0,  # Total bytes consumed by records (including prefixes and padding)
        'padding_bytes': 0,  # Bytes that are padding/zeros between records
        'unaccounted_bytes': 0,  # Bytes in valid_data_bytes that aren't accounted for
        'zeros_after_last_record': 0,  # Zeros found after the last valid record
        'zeros_between_records': 0,  # Zeros found between valid records
        'zeros_before_first_record': 0,  # Zeros found before the first record
        'last_record_end_offset': 0,  # Offset where the last record ends
        'zero_regions': []  # List of (start, end) offsets where zeros appear
    }
    
    if valid_data_bytes == 0 or capacity < HEADER_SIZE:
        # Skip empty or invalid frames
        if capacity > HEADER_SIZE:
            f.read(capacity - HEADER_SIZE)
        return (0, [], analysis)
    
    # Read data section (capacity - HEADER_SIZE bytes)
    data_size = capacity - HEADER_SIZE
    data = f.read(data_size)
    
    if len(data) < data_size:
        # File truncated
        return (0, [], analysis)
    
    # Parse records: each record has 4-byte length prefix + data
    record_count = 0
    record_sizes = []
    offset = 0
    first_record_start = None
    last_record_end = None
    
    while offset < valid_data_bytes:
        # Check if we have enough bytes for length prefix
        if offset + LENGTH_PREFIX_SIZE > valid_data_bytes:
            break
        
        # Read record length
        rec_len = struct.unpack('<I', data[offset:offset+LENGTH_PREFIX_SIZE])[0]
        offset += LENGTH_PREFIX_SIZE
        analysis['total_length_prefix_bytes'] += LENGTH_PREFIX_SIZE
        
        # Skip zero-length records (padding)
        if rec_len == 0:
            analysis['zero_length_records'] += 1
            # asyncloguploader format has NO alignment padding between records
            continue
        
        # Check if we have enough bytes for the record
        if offset + rec_len > valid_data_bytes:
            # Record extends beyond valid data - invalid frame
            break
        
        # Valid record found
        if first_record_start is None:
            first_record_start = offset - LENGTH_PREFIX_SIZE  # Include length prefix
        
        record_start = offset - LENGTH_PREFIX_SIZE  # Start of this record (including length prefix)
        record_count += 1
        if collect_sizes:
            record_sizes.append(rec_len)
        
        analysis['total_record_bytes'] += rec_len
        offset += rec_len
        # asyncloguploader format: records are packed back-to-back with NO alignment padding
        last_record_end = offset
    
    analysis['last_record_end_offset'] = last_record_end if last_record_end else 0
    
    # Now scan the buffer to find zero regions
    if analyze_padding and valid_data_bytes > 0:
        # Scan for zero regions in the valid_data_bytes range
        zero_start = None
        i = 0
        while i < valid_data_bytes:
            if data[i] == 0:
                if zero_start is None:
                    zero_start = i
            else:
                if zero_start is not None:
                    # End of zero region
                    zero_end = i
                    zero_size = zero_end - zero_start
                    if zero_size >= 4:  # Only track significant zero regions (>= 4 bytes)
                        analysis['zero_regions'].append((zero_start, zero_end))
                        
                        # Categorize zero regions
                        if first_record_start is None:
                            # No records found, all zeros are "before first record"
                            analysis['zeros_before_first_record'] += zero_size
                        elif zero_start < first_record_start:
                            # Zeros before first record
                            analysis['zeros_before_first_record'] += min(zero_size, first_record_start - zero_start)
                        elif last_record_end and zero_start >= last_record_end:
                            # Zeros after last record
                            analysis['zeros_after_last_record'] += zero_size
                        else:
                            # Zeros between records (or overlapping)
                            analysis['zeros_between_records'] += zero_size
                    zero_start = None
            i += 1
        
        # Handle trailing zeros
        if zero_start is not None:
            zero_end = valid_data_bytes
            zero_size = zero_end - zero_start
            if zero_size >= 4:
                analysis['zero_regions'].append((zero_start, zero_end))
                if last_record_end and zero_start >= last_record_end:
                    analysis['zeros_after_last_record'] += zero_size
                else:
                    analysis['zeros_between_records'] += zero_size
    
    # Calculate total consumed bytes
    analysis['total_consumed_bytes'] = (
        analysis['total_length_prefix_bytes'] + 
        analysis['total_record_bytes'] + 
        analysis['padding_bytes']
    )
    
    # Calculate unaccounted bytes (the gap between valid_data_bytes and what we accounted for)
    analysis['unaccounted_bytes'] = valid_data_bytes - analysis['total_consumed_bytes']
    
    return (record_count, record_sizes, analysis)


def count_entities_in_file(log_file: Path, verbose: bool = False) -> Tuple[int, int, int]:
    """
    Count entities (records) in a log file.
    
    Returns:
        (total_records, total_frames, frames_with_data)
    """
    total_records = 0
    total_frames = 0
    frames_with_data = 0
    
    with open(log_file, 'rb') as f:
        frame_num = 0
        
        while True:
            frame_start = f.tell()
            header_result = read_frame_header(f)
            
            if header_result is None:
                # End of file
                break
            
            capacity, valid_data_bytes = header_result
            frame_num += 1
            total_frames += 1
            
            if verbose:
                print(f"Frame {frame_num} @ {frame_start:,}: "
                      f"capacity={capacity:,} bytes ({capacity/(1024*1024):.2f}MB), "
                      f"valid_data={valid_data_bytes:,} bytes ({valid_data_bytes/(1024*1024):.2f}MB)",
                      file=sys.stderr)
            
            # Validate header
            if capacity < HEADER_SIZE:
                if verbose:
                    print(f"  [WARNING] Invalid capacity {capacity} < {HEADER_SIZE}, skipping frame",
                          file=sys.stderr)
                continue
            
            if valid_data_bytes > capacity:
                if verbose:
                    print(f"  [WARNING] valid_data_bytes ({valid_data_bytes}) > capacity ({capacity}), skipping frame",
                          file=sys.stderr)
                # Skip the frame data
                if capacity > HEADER_SIZE:
                    f.read(capacity - HEADER_SIZE)
                continue
            
            # Count records in this frame
            if valid_data_bytes > 0:
                frames_with_data += 1
                record_count, record_sizes, analysis = count_records_in_frame(
                    f, capacity, valid_data_bytes, 
                    collect_sizes=verbose,
                    analyze_padding=verbose
                )
                total_records += record_count
                
                if verbose:
                    print(f"  -> {record_count} records", file=sys.stderr)
                    if record_sizes:
                        # Format record sizes nicely
                        if len(record_sizes) <= 20:
                            # Show all sizes if 20 or fewer
                            sizes_str = ", ".join(f"{s:,}" for s in record_sizes)
                            print(f"    Record sizes: {sizes_str} bytes", file=sys.stderr)
                        else:
                            # Show first 10, summary, and last 10
                            first_10 = ", ".join(f"{s:,}" for s in record_sizes[:10])
                            last_10 = ", ".join(f"{s:,}" for s in record_sizes[-10:])
                            min_size = min(record_sizes)
                            max_size = max(record_sizes)
                            avg_size = sum(record_sizes) / len(record_sizes)
                            print(f"    Record sizes (first 10): {first_10} ...", file=sys.stderr)
                            print(f"    Record sizes (last 10): ... {last_10}", file=sys.stderr)
                            print(f"    Size stats: min={min_size:,} bytes, max={max_size:,} bytes, "
                                  f"avg={avg_size:,.0f} bytes ({avg_size/1024:.2f} KB)", file=sys.stderr)
                    
                    # Show padding analysis
                    print(f"    Data analysis:", file=sys.stderr)
                    print(f"      Total record data: {analysis['total_record_bytes']:,} bytes ({analysis['total_record_bytes']/(1024*1024):.2f} MB)", file=sys.stderr)
                    print(f"      Length prefix overhead: {analysis['total_length_prefix_bytes']:,} bytes ({analysis['total_length_prefix_bytes']/(1024*1024):.2f} MB)", file=sys.stderr)
                    print(f"      Padding/alignment bytes: {analysis['padding_bytes']:,} bytes ({analysis['padding_bytes']/(1024*1024):.2f} MB)", file=sys.stderr)
                    print(f"      Zero-length records: {analysis['zero_length_records']:,}", file=sys.stderr)
                    print(f"      Total consumed: {analysis['total_consumed_bytes']:,} bytes ({analysis['total_consumed_bytes']/(1024*1024):.2f} MB)", file=sys.stderr)
                    print(f"      Valid data bytes: {valid_data_bytes:,} bytes ({valid_data_bytes/(1024*1024):.2f} MB)", file=sys.stderr)
                    
                    # Show zero padding location analysis
                    if analysis['zeros_after_last_record'] > 0 or analysis['zeros_between_records'] > 0 or analysis['zeros_before_first_record'] > 0:
                        print(f"    Zero padding location:", file=sys.stderr)
                        if analysis['zeros_before_first_record'] > 0:
                            print(f"      Before first record: {analysis['zeros_before_first_record']:,} bytes ({analysis['zeros_before_first_record']/(1024*1024):.2f} MB)", file=sys.stderr)
                        if analysis['zeros_between_records'] > 0:
                            print(f"      Between records: {analysis['zeros_between_records']:,} bytes ({analysis['zeros_between_records']/(1024*1024):.2f} MB)", file=sys.stderr)
                        if analysis['zeros_after_last_record'] > 0:
                            print(f"      After last record: {analysis['zeros_after_last_record']:,} bytes ({analysis['zeros_after_last_record']/(1024*1024):.2f} MB) ⚠️", file=sys.stderr)
                        if analysis['last_record_end_offset'] > 0:
                            print(f"      Last record ends at offset: {analysis['last_record_end_offset']:,} bytes", file=sys.stderr)
                            print(f"      Gap to valid_data_bytes: {valid_data_bytes - analysis['last_record_end_offset']:,} bytes ({((valid_data_bytes - analysis['last_record_end_offset'])/(1024*1024)):.2f} MB)", file=sys.stderr)
                        
                        # Show zero regions if there are significant ones
                        if len(analysis['zero_regions']) > 0 and len(analysis['zero_regions']) <= 20:
                            print(f"      Zero regions (start, end):", file=sys.stderr)
                            for start, end in analysis['zero_regions'][:10]:  # Show first 10
                                size = end - start
                                print(f"        [{start:,}, {end:,}) = {size:,} bytes ({size/(1024*1024):.2f} MB)", file=sys.stderr)
                            if len(analysis['zero_regions']) > 10:
                                print(f"        ... and {len(analysis['zero_regions']) - 10} more regions", file=sys.stderr)
                    
                    if analysis['unaccounted_bytes'] > 0:
                        print(f"      ⚠️  Unaccounted bytes: {analysis['unaccounted_bytes']:,} bytes ({analysis['unaccounted_bytes']/(1024*1024):.2f} MB) - likely zeros/padding!", file=sys.stderr)
                    elif analysis['unaccounted_bytes'] < 0:
                        print(f"      ⚠️  Over-consumed bytes: {abs(analysis['unaccounted_bytes']):,} bytes - parsing error!", file=sys.stderr)
            else:
                # Empty frame - skip data section
                if verbose:
                    print(f"  -> Empty frame (no records)", file=sys.stderr)
                if capacity > HEADER_SIZE:
                    f.read(capacity - HEADER_SIZE)
    
    return (total_records, total_frames, frames_with_data)


def main():
    import argparse
    
    parser = argparse.ArgumentParser(
        description='Count entities (records) in asyncloguploader log files',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s logfile.log
  %(prog)s logfile.log --verbose
  %(prog)s logfile.log --json
        """
    )
    
    parser.add_argument(
        'log_file',
        type=Path,
        help='Path to the log file to analyze'
    )
    
    parser.add_argument(
        '-v', '--verbose',
        action='store_true',
        help='Show detailed frame information including record sizes'
    )
    
    parser.add_argument(
        '--json',
        action='store_true',
        help='Output results in JSON format'
    )
    
    args = parser.parse_args()
    
    if not args.log_file.exists():
        print(f"Error: File not found: {args.log_file}", file=sys.stderr)
        sys.exit(1)
    
    file_size = args.log_file.stat().st_size
    
    if args.verbose:
        print(f"Analyzing file: {args.log_file}", file=sys.stderr)
        print(f"File size: {file_size:,} bytes ({file_size/(1024*1024):.2f} MB)", file=sys.stderr)
        print(file=sys.stderr)
    
    try:
        total_records, total_frames, frames_with_data = count_entities_in_file(
            args.log_file, 
            verbose=args.verbose
        )
        
        if args.json:
            result = {
                'file': str(args.log_file),
                'file_size_bytes': file_size,
                'file_size_mb': round(file_size / (1024 * 1024), 2),
                'total_records': total_records,
                'total_frames': total_frames,
                'frames_with_data': frames_with_data,
                'empty_frames': total_frames - frames_with_data,
                'avg_records_per_frame': round(total_records / frames_with_data, 2) if frames_with_data > 0 else 0
            }
            print(json.dumps(result, indent=2))
        else:
            print(f"Total entities (records): {total_records:,}")
            print(f"Total frames: {total_frames:,}")
            print(f"Frames with data: {frames_with_data:,}")
            print(f"Empty frames: {total_frames - frames_with_data:,}")
            if frames_with_data > 0:
                print(f"Average records per frame: {total_records / frames_with_data:.2f}")
            print(f"File size: {file_size:,} bytes ({file_size/(1024*1024):.2f} MB)")
            
            if total_records > 0:
                bytes_per_record = file_size / total_records
                print(f"Average bytes per record: {bytes_per_record:,.0f} bytes ({bytes_per_record/1024:.2f} KB)")
    
    except Exception as e:
        print(f"Error processing file: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()