"""I/O operations for schema fetching and protobuf parsing."""

import os
import json
import urllib.request
import urllib.parse
import urllib.error
from typing import Optional

from .types import DecodedMPLog, FeatureInfo
from .decoder import ByteReader, read_varint, skip_field
from .utils import unpack_metadata_byte


# Schema cache to avoid repeated API calls
# Cache key: (model_config_id, version)
_schema_cache: dict[tuple[str, int], list[FeatureInfo]] = {}


def get_feature_schema(model_config_id: str, version: int, inference_host: Optional[str] = None, api_path: Optional[str] = None) -> list[FeatureInfo]:
    """Fetch feature schema from inference API with caching.
    
    Results are cached per (model_config_id, version) combination
    to avoid repeated API calls for the same model proxy and version.
    
    Args:
        model_config_id: The model proxy config ID
        version: The schema version
        inference_host: Inference service host URL. If None, reads from INFERENCE_HOST env var.
        api_path: API path. If None, reads from INFERENCE_PATH env var or uses default.
    """
    # Read from environment variables if not provided
    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")
    
    if api_path is None:
        api_path = os.getenv("INFERENCE_PATH", "/api/v1/inference/mp-config-registry/get_feature_schema")
    
    cache_key = (model_config_id, version)
    if cache_key in _schema_cache:
        return _schema_cache[cache_key]
    
    base_url = f"{inference_host}{api_path}"
    params = urllib.parse.urlencode({
        "model_config_id": model_config_id,
        "version": str(version)
    })
    url = f"{base_url}?{params}"
    
    req = urllib.request.Request(url, headers={"Content-Type": "application/json"})
    
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            data = json.loads(response.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8') if e.fp else str(e)
        raise Exception(f"HTTP Error {e.code} from inference service: {error_body}")
    except urllib.error.URLError as e:
        raise Exception(f"URL Error connecting to inference service: {e.reason}")
    
    features = []
    for idx, component in enumerate(data.get("data", [])):
        features.append(FeatureInfo(
            name=component["feature_name"],
            feature_type=component["feature_type"].upper(),
            index=idx
        ))
    
    if not features:
        raise Exception(f"No features found in schema for model_config_id={model_config_id}, version={version}")
    
    _schema_cache[cache_key] = features
    return features


def parse_mplog_protobuf(data: bytes) -> DecodedMPLog:
    """Parse the outer MPLog protobuf message."""
    result = DecodedMPLog()
    encoded_features_list: list[bytes] = []
    
    reader = ByteReader(data)
    
    while reader.has_more():
        tag = read_varint(reader)
        field_number = tag >> 3
        wire_type = tag & 0x7
        
        if field_number == 1 and wire_type == 2:  # user_id
            length = read_varint(reader)
            result.user_id = reader.read(length).decode('utf-8')
        
        elif field_number == 2 and wire_type == 2:  # tracking_id
            length = read_varint(reader)
            result.tracking_id = reader.read(length).decode('utf-8')
        
        elif field_number == 3 and wire_type == 2:  # mp_config_id
            length = read_varint(reader)
            result.model_proxy_config_id = reader.read(length).decode('utf-8')
        
        elif field_number == 4 and wire_type == 2:  # entities (repeated string)
            length = read_varint(reader)
            result.entities.append(reader.read(length).decode('utf-8'))
        
        elif field_number == 5 and wire_type == 2:  # features (repeated perEntityFeatures)
            length = read_varint(reader)
            feature_bytes = reader.read(length)
            # Parse perEntityFeatures message
            per_entity_reader = ByteReader(feature_bytes)
            while per_entity_reader.has_more():
                inner_tag = read_varint(per_entity_reader)
                inner_field = inner_tag >> 3
                inner_wire = inner_tag & 0x7
                if inner_field == 1 and inner_wire == 2:  # encoded_features
                    enc_len = read_varint(per_entity_reader)
                    encoded_features_list.append(per_entity_reader.read(enc_len))
                else:
                    skip_field(per_entity_reader, inner_wire)
        
        elif field_number == 6 and wire_type == 2:  # metadata
            length = read_varint(reader)
            metadata_bytes = reader.read(length)
            if len(metadata_bytes) > 0:
                result.metadata_byte = metadata_bytes[0]
                result.compression_enabled, result.version, result.format_type = unpack_metadata_byte(result.metadata_byte)
        
        elif field_number == 7 and wire_type == 2:  # parent_entity (repeated string)
            length = read_varint(reader)
            result.parent_entity.append(reader.read(length).decode('utf-8'))
        
        else:
            skip_field(reader, wire_type)
    
    # Attach encoded features for later processing
    result._encoded_features = encoded_features_list
    return result


def get_mplog_metadata(log_data: bytes, decompress: bool = True) -> DecodedMPLog:
    """
    Extract metadata from MPLog bytes without full decoding.
    
    Args:
        log_data: The MPLog bytes (possibly compressed)
        decompress: Whether to attempt zstd decompression
    
    Returns:
        DecodedMPLog with metadata fields populated
    """
    import zstandard as zstd
    
    working_data = log_data
    if decompress:
        try:
            if len(log_data) >= 4 and log_data[:4] == b'\x28\xB5\x2F\xFD':
                decompressor = zstd.ZstdDecompressor()
                working_data = decompressor.decompress(log_data)
        except Exception:
            working_data = log_data
    
    return parse_mplog_protobuf(working_data)
