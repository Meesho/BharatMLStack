"""
gRPC Feature Client SDK for BharatML Stack

A high-performance gRPC client for feature operations including persist, retrieve, and retrieveDecoded.
"""

from .client import GRPCFeatureClient
from .config import GRPCClientConfig

__version__ = "0.1.0"
__all__ = ["GRPCFeatureClient", "GRPCClientConfig"] 