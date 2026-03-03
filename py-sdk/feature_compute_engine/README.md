# Feature Compute Engine

Asset-based feature computation platform SDK for BharatML Stack.

## Installation

```bash
pip install feature_compute_engine
```

## Quick Start

```python
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
```
