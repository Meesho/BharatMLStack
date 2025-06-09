"""
Feature processing utilities for BharatML Stack SDKs
"""

from typing import Dict, List, Tuple, Any


def get_fgs_to_feature_mappings(feature_group_kv: Dict[str, List[Tuple[str, str]]]) -> Tuple[Dict[str, List[str]], Dict[str, List[str]]]:
    """
    Generate feature group to feature name mappings
    
    Args:
        feature_group_kv: Dictionary mapping feature group labels to list of (offline_feat_name, online_feat_name) tuples
        
    Returns:
        Tuple of (onfs_fg_to_onfs_feat_map, onfs_fg_to_ofs_feat_map)
        - onfs_fg_to_onfs_feat_map: Maps feature group to online feature names
        - onfs_fg_to_ofs_feat_map: Maps feature group to offline feature names
    """
    onfs_fg_to_onfs_feat_map: Dict[str, List[str]] = {}  # onfs fg -> online feat name
    for fg in feature_group_kv:
        if fg not in onfs_fg_to_onfs_feat_map:
            onfs_fg_to_onfs_feat_map[fg] = []

        for onfs_feat_name in feature_group_kv[fg]:
            onfs_fg_to_onfs_feat_map[fg].append(onfs_feat_name[1])

    onfs_fg_to_ofs_feat_map: Dict[str, List[str]] = {}  # onfs fg -> offline feat name
    for fg in feature_group_kv:
        if fg not in onfs_fg_to_ofs_feat_map:
            onfs_fg_to_ofs_feat_map[fg] = []

        for onfs_feat_name in feature_group_kv[fg]:
            onfs_fg_to_ofs_feat_map[fg].append(onfs_feat_name[0])
            
    return onfs_fg_to_onfs_feat_map, onfs_fg_to_ofs_feat_map


def extract_entity_info(json_response: Dict[str, Any]) -> Tuple[str, List[str]]:
    """
    Extract entity label and column names from metadata response
    
    Args:
        json_response: JSON response from metadata API
        
    Returns:
        Tuple of (entity_label, entity_column_names)
    """
    entity_label = list(json_response["keys"].keys())[0]
    entity_column_names = list(json_response["keys"].values())[0]
    return entity_label, entity_column_names 