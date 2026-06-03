#!/usr/bin/env python3
"""
Count all entity IDs in a parsed log file, including duplicates.
"""
import re
import sys
from pathlib import Path
from collections import Counter


def count_entity_ids(parsed_log_path: Path):
    """
    Count all entity IDs from the parsed log file.
    Extracts from both:
    1. "Entity IDs: ['id1', 'id2', ...]" lines
    2. "Entity ID: <id>" lines
    """
    entity_ids = []
    
    with open(parsed_log_path, 'r') as f:
        for line in f:
            # Match "Entity IDs: ['id1', 'id2', ...]"
            match = re.search(r"Entity IDs:\s*\[(.*?)\]", line)
            if match:
                ids_str = match.group(1)
                # Extract all quoted IDs
                ids = re.findall(r"'(\d+)'", ids_str)
                entity_ids.extend(ids)
            
            # Also match individual "Entity ID: <id>" lines
            match = re.search(r"^Entity ID:\s*(\d+)$", line)
            if match:
                entity_ids.append(match.group(1))
    
    return entity_ids


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 count_entity_ids.py <parsed_log_file>")
        sys.exit(1)
    
    parsed_log_path = Path(sys.argv[1])
    if not parsed_log_path.exists():
        print(f"Error: File not found: {parsed_log_path}")
        sys.exit(1)
    
    print(f"Counting entity IDs from: {parsed_log_path}")
    print("This may take a moment for large files...")
    
    entity_ids = count_entity_ids(parsed_log_path)
    
    total_count = len(entity_ids)
    unique_count = len(set(entity_ids))
    duplicate_count = total_count - unique_count
    
    # Count frequency of each ID
    id_counter = Counter(entity_ids)
    
    print(f"\n=== Entity ID Statistics ===")
    print(f"Total entity IDs (including duplicates): {total_count:,}")
    print(f"Unique entity IDs: {unique_count:,}")
    print(f"Duplicate entity IDs: {duplicate_count:,}")
    
    if duplicate_count > 0:
        print(f"\nTop 10 most frequent entity IDs:")
        for entity_id, count in id_counter.most_common(10):
            print(f"  {entity_id}: {count} occurrences")
    
    return 0


if __name__ == "__main__":
    sys.exit(main())



