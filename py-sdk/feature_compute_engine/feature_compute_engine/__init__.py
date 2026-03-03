"""
Feature Compute Engine — Asset-based feature computation platform SDK.

Usage:
    from feature_compute_engine import asset, Input

    @asset(
        name="fg.user_spend",
        entity="user",
        entity_key="user_id",
        schedule="3h",
        inputs=[Input("silver.orders")],
    )
    def user_spend(ctx, ds, orders_df):
        return orders_df.groupBy("user_id").agg(...)
"""
from .types.asset_spec import AssetSpec, Input
from .types.necessity import ComputeKey, Necessity
from .types.execution_plan import AssetExecutionPlan, NotebookExecutionPlan
from .decorators.asset import asset
from .decorators.registry import AssetRegistry
from .runtime.context import AssetContext, DevContext
from .runtime.result import AssetResult, NotebookRunResult
from .runtime.notebook_runtime import NotebookRuntime
from .client.horizon_client import HorizonClient, HorizonClientConfig, HorizonClientError
from .client.airflow_trigger import AirflowTrigger, AirflowTriggerConfig

__version__ = "0.1.0"

__all__ = [
    "asset",
    "Input",
    "AssetSpec",
    "Necessity",
    "ComputeKey",
    "NotebookExecutionPlan",
    "AssetExecutionPlan",
    "AssetRegistry",
    "AssetContext",
    "DevContext",
    "AssetResult",
    "NotebookRunResult",
    "NotebookRuntime",
    "HorizonClient",
    "HorizonClientConfig",
    "HorizonClientError",
    "AirflowTrigger",
    "AirflowTriggerConfig",
]
