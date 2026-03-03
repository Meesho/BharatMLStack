#!/usr/bin/env python3
import re
import sys
from collections import Counter

parsed_log = sys.argv[1] if len(sys.argv) > 1 else 'Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.parsed.log'

entity_ids = []

print(f"Processing {parsed_log}...")
with open(parsed_log, 'r') as f:
    for line_num, line in enumerate(f, 1):
        # Extract from "Entity IDs: ['id1', 'id2', ...]" (not "Parent Entity IDs")
        if line.startswith("Entity IDs:") and not line.startswith("Parent Entity IDs:"):
            match = re.search(r"Entity IDs:\s*\[(.*?)\]", line)
            if match:
                ids_str = match.group(1)
                ids = re.findall(r"'(\d+)'", ids_str)
                entity_ids.extend(ids)
        
        if line_num % 10000 == 0:
            print(f"  Processed {line_num:,} lines, found {len(entity_ids):,} entity IDs so far...", file=sys.stderr)

total = len(entity_ids)
unique = len(set(entity_ids))
duplicates = total - unique

print(f"Total entity IDs (including duplicates): {total:,}")
print(f"Unique entity IDs: {unique:,}")
print(f"Duplicate entity IDs: {duplicates:,}")

if duplicates > 0:
    counter = Counter(entity_ids)
    print(f"\nTop 10 most frequent entity IDs:")
    for eid, cnt in counter.most_common(10):
        print(f"  {eid}: {cnt} occurrences")

