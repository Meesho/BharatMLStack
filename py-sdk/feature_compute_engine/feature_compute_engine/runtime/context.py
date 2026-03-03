"""
Runtime context passed to every asset compute function.

Provides partition info, watermarks, and metadata.
Does NOT provide DataFrames — those are passed as function arguments.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, Optional


@dataclass
class AssetContext:
    """
    Context object passed as the first argument to every asset function.

    Usage::

        @asset(...)
        def my_asset(ctx: AssetContext, ds: str, orders_df):
            print(ctx.partition)     # "2026-03-01"
            print(ctx.asset_name)    # "fg.user_spend"
            print(ctx.compute_key)   # "a1b2c3d4"
    """

    partition: str
    asset_name: str
    compute_key: str
    artifact_path: str
    necessity: str
    input_versions: Dict[str, str] = field(default_factory=dict)
    run_metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class DevContext:
    """
    Simplified context for interactive notebook development.

    Usage::

        ctx = DevContext(partition="2026-03-01")
        result = my_asset(ctx, "2026-03-01", orders_df)
    """

    partition: str
    asset_name: str = "dev"
    compute_key: str = "dev"
    artifact_path: str = "/tmp/dev"
    necessity: str = "active"
    input_versions: Dict[str, str] = field(default_factory=dict)
    run_metadata: Dict[str, Any] = field(default_factory=dict)
