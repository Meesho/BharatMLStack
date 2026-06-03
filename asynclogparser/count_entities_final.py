#!/usr/bin/env python3
"""Count entity IDs from parsed log file."""
import re
import sys
from collections import Counter

def main():
    if len(sys.argv) < 2:
        filename = 'Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.parsed.log'
    else:
        filename = sys.argv[1]
    entity_ids = []
    
    sys.stderr.write(f"Processing {filename}...\n")
    sys.stderr.flush()
    
    try:
        with open(filename, 'r', encoding='utf-8') as f:
            for line_num, line in enumerate(f, 1):
                if line.startswith("Entity IDs:") and not line.startswith("Parent Entity IDs:"):
                    match = re.search(r"Entity IDs:\s*\[(.*?)\]", line)
                    if match:
                        ids_str = match.group(1)
                        ids = re.findall(r"'(\d+)'", ids_str)
                        entity_ids.extend(ids)
                
                if line_num % 5000 == 0:
                    sys.stderr.write(f"  Line {line_num:,}, IDs so far: {len(entity_ids):,}\n")
                    sys.stderr.flush()
        
        total = len(entity_ids)
        unique = len(set(entity_ids))
        duplicates = total - unique
        
        result = f"""Total entity IDs (including duplicates): {total:,}
Unique entity IDs: {unique:,}
Duplicate entity IDs: {duplicates:,}
"""
        
        if duplicates > 0:
            counter = Counter(entity_ids)
            result += "\nTop 10 most frequent entity IDs:\n"
            for eid, cnt in counter.most_common(10):
                result += f"  {eid}: {cnt} occurrences\n"
        
        sys.stdout.write(result)
        sys.stdout.flush()
        
        # Also write to file
        with open('entity_count_result.txt', 'w') as out:
            out.write(result)
        
        return 0
    except Exception as e:
        sys.stderr.write(f"Error: {e}\n")
        import traceback
        traceback.print_exc(file=sys.stderr)
        return 1

if __name__ == '__main__':
    sys.exit(main())



