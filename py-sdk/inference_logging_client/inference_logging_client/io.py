"""I/O operations for schema fetching and protobuf parsing."""

import os
import sys
import json
import time
import threading
import urllib.request
import urllib.parse
import urllib.error
from typing import Optional
from collections import OrderedDict

from .types import DecodedMPLog, FeatureInfo
from .decoder import ByteReader, read_varint, skip_field
from .utils import unpack_metadata_byte
from .exceptions import SchemaFetchError, SchemaNotFoundError


# Thread-safe LRU cache for schemas with max size limit
_schema_cache: OrderedDict[tuple[str, int], list[FeatureInfo]] = OrderedDict()
_schema_cache_lock = threading.Lock()
_SCHEMA_CACHE_MAX_SIZE = 100  # Maximum number of cached schemas

# Retry configuration
_MAX_RETRIES = 3
_RETRY_BACKOFF_BASE = 0.5  # seconds


def _fetch_schema_with_retry(url: str, max_retries: int = _MAX_RETRIES) -> dict:
    """Fetch schema from URL with exponential backoff retry.
    
    Args:
        url: The URL to fetch from
        max_retries: Maximum number of retry attempts
        
    Returns:
        Parsed JSON response as dict
        
    Raises:
        SchemaFetchError: If all retries fail
    """
    last_exception = None
    
    for attempt in range(max_retries):
        try:
            req = urllib.request.Request(url, headers={
                "Content-Type": "application/json",
                "User-Agent": "inference-logging-client/0.1.0"
            })
            with urllib.request.urlopen(req, timeout=30) as response:
                return json.loads(response.read().decode('utf-8'))
        except urllib.error.HTTPError as e:
            error_body = e.read().decode('utf-8') if e.fp else str(e)
            # Don't retry on client errors (4xx)
            if 400 <= e.code < 500:
                raise SchemaFetchError(f"HTTP Error {e.code} from inference service: {error_body}")
            last_exception = SchemaFetchError(f"HTTP Error {e.code} from inference service: {error_body}")
        except urllib.error.URLError as e:
            last_exception = SchemaFetchError(f"URL Error connecting to inference service: {e.reason}")
        except Exception as e:
            last_exception = SchemaFetchError(f"Unexpected error fetching schema: {e}")
        
        # Exponential backoff before retry
        if attempt < max_retries - 1:
            sleep_time = _RETRY_BACKOFF_BASE * (2 ** attempt)
            time.sleep(sleep_time)
    
    raise last_exception


def get_feature_schema(model_config_id: str, version: int, inference_host: Optional[str] = None, api_path: Optional[str] = None) -> list[FeatureInfo]:
    """Fetch feature schema from inference API with caching.
    
    Results are cached per (model_config_id, version) combination
    to avoid repeated API calls for the same model proxy and version.
    Cache is thread-safe and has a maximum size limit with LRU eviction.
    
    Args:
        model_config_id: The model proxy config ID
        version: The schema version
        inference_host: Inference service host URL. If None, reads from INFERENCE_HOST env var.
        api_path: API path. If None, reads from INFERENCE_PATH env var or uses default.
        
    Raises:
        SchemaFetchError: If schema fetch fails after retries
        SchemaNotFoundError: If no features found in schema
    """
    # Read from environment variables if not provided
    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")
    
    if api_path is None:
        api_path = os.getenv("INFERENCE_PATH", "/api/v1/inference/mp-config-registry/get_feature_schema")
    
    cache_key = (model_config_id, version)
    
    # Thread-safe cache lookup
    with _schema_cache_lock:
        if cache_key in _schema_cache:
            # Move to end (most recently used)
            _schema_cache.move_to_end(cache_key)
            return _schema_cache[cache_key]
    
    base_url = f"{inference_host}{api_path}"
    params = urllib.parse.urlencode({
        "model_config_id": model_config_id,
        "version": str(version)
    })
    url = f"{base_url}?{params}"
    
    # Fetch with retry
    data = _fetch_schema_with_retry(url)
    
    features = []
    for idx, component in enumerate(data.get("data", [])):
        features.append(FeatureInfo(
            name=component["feature_name"],
            feature_type=component["feature_type"].upper(),
            index=idx
        ))
    
    if not features:
        raise SchemaNotFoundError(f"No features found in schema for model_config_id={model_config_id}, version={version}")
    
    # Thread-safe cache update with LRU eviction
    with _schema_cache_lock:
        # Evict oldest entries if at capacity
        while len(_schema_cache) >= _SCHEMA_CACHE_MAX_SIZE:
            _schema_cache.popitem(last=False)
        _schema_cache[cache_key] = features
    
    return features


def clear_schema_cache() -> None:
    """Clear the schema cache. Useful for testing or when schemas have changed."""
    with _schema_cache_lock:
        _schema_cache.clear()


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
        
    Raises:
        ImportError: If data is zstd-compressed but zstandard is not installed
    """
    working_data = log_data
    if decompress:
        # Check for zstd magic number: 0x28 0xB5 0x2F 0xFD
        if len(log_data) >= 4 and log_data[:4] == b'\x28\xB5\x2F\xFD':
            try:
                import zstandard as zstd
                decompressor = zstd.ZstdDecompressor()
                working_data = decompressor.decompress(log_data)
            except ImportError:
                raise ImportError(
                    "Data appears to be zstd-compressed but the 'zstandard' package is not installed. "
                    "Install it with: pip install zstandard"
                )
    
    return parse_mplog_protobuf(working_data)
