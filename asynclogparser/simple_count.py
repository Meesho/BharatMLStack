#!/usr/bin/env python3
import re
import sys
from collections import Counter

try:
    entity_ids = []
    filename = 'Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.parsed.log'
    
    print(f"Opening file: {filename}", file=sys.stderr)
    with open(filename, 'r') as f:
        line_count = 0
        for line in f:
            line_count += 1
            if line.startswith("Entity IDs:") and not line.startswith("Parent Entity IDs:"):
                match = re.search(r"Entity IDs:\s*\[(.*?)\]", line)
                if match:
                    ids_str = match.group(1)
                    ids = re.findall(r"'(\d+)'", ids_str)
                    entity_ids.extend(ids)
            if line_count % 10000 == 0:
                print(f"Processed {line_count:,} lines, found {len(entity_ids):,} entity IDs so far...", file=sys.stderr)
    
    print(f"Finished reading file. Total lines processed: {line_count:,}", file=sys.stderr)
    
    total = len(entity_ids)
    unique = len(set(entity_ids))
    duplicates = total - unique
    
    with open('entity_count_result.txt', 'w') as out:
        out.write(f"Total entity IDs (including duplicates): {total:,}\n")
        out.write(f"Unique entity IDs: {unique:,}\n")
        out.write(f"Duplicate entity IDs: {duplicates:,}\n")
        
        if duplicates > 0:
            counter = Counter(entity_ids)
            out.write(f"\nTop 10 most frequent entity IDs:\n")
            for eid, cnt in counter.most_common(10):
                out.write(f"  {eid}: {cnt} occurrences\n")
    
    # Also print to stdout
    print(f"Total entity IDs (including duplicates): {total:,}")
    print(f"Unique entity IDs: {unique:,}")
    print(f"Duplicate entity IDs: {duplicates:,}")
    
    if duplicates > 0:
        counter = Counter(entity_ids)
        print(f"\nTop 10 most frequent entity IDs:")
        for eid, cnt in counter.most_common(10):
            print(f"  {eid}: {cnt} occurrences")
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    import traceback
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)

