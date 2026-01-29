"""Type definitions for inference-logging-client."""

from enum import Enum
from dataclasses import dataclass, field


class Format(Enum):
    """Supported log formats."""
    PROTO = "proto"
    ARROW = "arrow"
    PARQUET = "parquet"


# Format type constants matching Go encoder (bits 6-7 of metadata byte)
FORMAT_TYPE_PROTO = 0    # 00
FORMAT_TYPE_ARROW = 1    # 01
FORMAT_TYPE_PARQUET = 2  # 10

# Mapping from format type int to Format enum
FORMAT_TYPE_MAP = {
    FORMAT_TYPE_PROTO: Format.PROTO,
    FORMAT_TYPE_ARROW: Format.ARROW,
    FORMAT_TYPE_PARQUET: Format.PARQUET,
}


@dataclass
class FeatureInfo:
    """Feature schema information."""
    name: str
    feature_type: str
    index: int


@dataclass
class DecodedMPLog:
    """Container for decoded MPLog data."""
    user_id: str = ""
    tracking_id: str = ""
    model_proxy_config_id: str = ""
    entities: list[str] = field(default_factory=list)
    parent_entity: list[str] = field(default_factory=list)
    metadata_byte: int = 0
    compression_enabled: bool = False
    version: int = 0
    format_type: int = 0  # 0=proto, 1=arrow, 2=parquet
