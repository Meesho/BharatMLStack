"""Tests for the @asset decorator."""
from __future__ import annotations

import pytest

from feature_compute_engine.decorators.asset import asset
from feature_compute_engine.decorators.registry import AssetRegistry
from feature_compute_engine.types.asset_spec import Input


@pytest.fixture(autouse=True)
def _clean_registry():
    """Clear the global registry before each test."""
    AssetRegistry.clear()
    yield
    AssetRegistry.clear()


class TestAssetDecorator:
    def test_spec_attached_to_function(self) -> None:
        @asset(
            name="fg.test_asset",
            entity="user",
            entity_key="user_id",
            schedule="3h",
        )
        def my_compute(ctx, ds):
            return "result"

        assert hasattr(my_compute, "_asset_spec")
        assert my_compute._asset_spec.name == "fg.test_asset"
        assert my_compute._asset_spec.entity == "user"
        assert my_compute._asset_spec.entity_key == "user_id"
        assert my_compute._asset_spec.trigger == "schedule_3h"

    def test_decorated_function_still_callable(self) -> None:
        @asset(
            name="fg.callable_test",
            entity="user",
            entity_key="user_id",
            schedule="3h",
        )
        def my_compute(ctx, ds):
            return ds + "_processed"

        result = my_compute("ctx", "2026-03-01")
        assert result == "2026-03-01_processed"

    def test_function_name_preserved(self) -> None:
        @asset(
            name="fg.name_test",
            entity="user",
            entity_key="user_id",
            schedule="3h",
        )
        def user_spend(ctx, ds):
            pass

        assert user_spend.__name__ == "user_spend"

    def test_inputs_passed_through(self) -> None:
        inputs = [
            Input("silver.orders", partition="ds"),
            Input("silver.payments", partition="ds", window=7),
        ]

        @asset(
            name="fg.with_inputs",
            entity="user",
            entity_key="user_id",
            schedule="3h",
            inputs=inputs,
        )
        def my_compute(ctx, ds, orders, payments):
            pass

        spec = my_compute._asset_spec
        assert len(spec.inputs) == 2
        assert spec.inputs[0].name == "silver.orders"
        assert spec.inputs[1].window == 7

    def test_checks_passed_through(self) -> None:
        @asset(
            name="fg.with_checks",
            entity="user",
            entity_key="user_id",
            schedule="3h",
            checks=["row_count > 0", "null_pct < 0.01"],
        )
        def my_compute(ctx, ds):
            pass

        assert my_compute._asset_spec.checks == [
            "row_count > 0",
            "null_pct < 0.01",
        ]

    def test_multiple_assets_different_schedules(self) -> None:
        @asset(
            name="fg.fast_features",
            entity="user",
            entity_key="user_id",
            schedule="3h",
        )
        def fast(ctx, ds):
            pass

        @asset(
            name="fg.slow_features",
            entity="user",
            entity_key="user_id",
            schedule="24h",
        )
        def slow(ctx, ds):
            pass

        assert fast._asset_spec.trigger == "schedule_3h"
        assert slow._asset_spec.trigger == "schedule_24h"
        assert AssetRegistry.count() == 2

    def test_upstream_trigger_asset(self) -> None:
        @asset(
            name="fg.derived",
            entity="user",
            entity_key="user_id",
            trigger="upstream",
            inputs=[Input("fg.fast_features")],
        )
        def derived(ctx, ds, fast_df):
            pass

        spec = derived._asset_spec
        assert spec.trigger == "upstream"
        assert spec.schedule is None

    def test_decorator_registers_in_global_registry(self) -> None:
        @asset(
            name="fg.registered",
            entity="product",
            entity_key="product_id",
            schedule="6h",
        )
        def my_fn(ctx, ds):
            pass

        found = AssetRegistry.get("fg.registered")
        assert found.name == "fg.registered"
        assert found.entity == "product"

    def test_serving_defaults_to_true(self) -> None:
        @asset(
            name="fg.default_serving",
            entity="user",
            entity_key="user_id",
            schedule="3h",
        )
        def fn(ctx, ds):
            pass

        assert fn._asset_spec.serving is True

    def test_serving_false(self) -> None:
        @asset(
            name="fg.transient",
            entity="user",
            entity_key="user_id",
            schedule="3h",
            serving=False,
        )
        def fn(ctx, ds):
            pass

        assert fn._asset_spec.serving is False
