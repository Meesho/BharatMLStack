"""
The @asset decorator — the primary authoring interface for data scientists.

Usage in a notebook:

    @asset(
        name="fg.user_spend",
        entity="user",
        entity_key="user_id",
        schedule="3h",
        serving=True,
        inputs=[Input("silver.orders", partition="ds")],
        checks=["row_count > 0"],
    )
    def user_spend(ctx, ds, orders_df):
        return orders_df.groupBy("user_id").agg(...)
"""
from __future__ import annotations

from functools import wraps
from typing import Callable, List, Optional

from ..types.asset_spec import AssetSpec, Input
from .registry import AssetRegistry


def asset(
    name: str,
    entity: str,
    entity_key: str,
    partition: str = "ds",
    schedule: Optional[str] = None,
    trigger: str = "upstream",
    serving: bool = True,
    incremental: bool = False,
    freshness: Optional[str] = None,
    inputs: Optional[List[Input]] = None,
    checks: Optional[List[str]] = None,
) -> Callable:
    """
    Decorator that registers a function as a feature computation asset.

    The decorated function signature should be:
        def compute_fn(ctx, partition_value, input_df_1, input_df_2, ...):
            return output_dataframe

    Where:
        - ctx: runtime context (partition info, watermarks, etc.)
        - partition_value: the partition being computed (e.g. "2026-03-01")
        - input_df_N: one DataFrame per declared input, in declaration order
    """

    def decorator(fn: Callable) -> Callable:
        notebook = getattr(fn, "__module__", None) or "unknown"

        spec = AssetSpec(
            name=name,
            entity=entity,
            entity_key=entity_key,
            notebook=notebook,
            partition=partition,
            trigger=trigger,
            schedule=schedule,
            serving=serving,
            incremental=incremental,
            freshness=freshness,
            inputs=inputs or [],
            checks=checks or [],
            _compute_fn=fn,
        )

        AssetRegistry.register(spec)

        fn._asset_spec = spec  # type: ignore[attr-defined]

        @wraps(fn)
        def wrapper(*args, **kwargs):  # type: ignore[no-untyped-def]
            return fn(*args, **kwargs)

        wrapper._asset_spec = spec  # type: ignore[attr-defined]
        return wrapper

    return decorator
