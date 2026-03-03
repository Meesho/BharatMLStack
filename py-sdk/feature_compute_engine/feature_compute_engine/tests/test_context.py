"""Tests for runtime context types."""
from __future__ import annotations

from feature_compute_engine.runtime.context import AssetContext, DevContext


class TestAssetContext:
    def test_creation_with_all_fields(self) -> None:
        ctx = AssetContext(
            partition="2026-03-01",
            asset_name="fg.user_spend",
            compute_key="abc123",
            artifact_path="/_artifacts/fg.user_spend/2026-03-01/abc123/",
            necessity="active",
            input_versions={"silver.orders": "v42"},
            run_metadata={"env": "prod"},
        )
        assert ctx.partition == "2026-03-01"
        assert ctx.asset_name == "fg.user_spend"
        assert ctx.compute_key == "abc123"
        assert ctx.artifact_path == "/_artifacts/fg.user_spend/2026-03-01/abc123/"
        assert ctx.necessity == "active"
        assert ctx.input_versions == {"silver.orders": "v42"}
        assert ctx.run_metadata == {"env": "prod"}

    def test_default_dicts_are_empty(self) -> None:
        ctx = AssetContext(
            partition="2026-03-01",
            asset_name="fg.test",
            compute_key="key1",
            artifact_path="/tmp/out",
            necessity="transient",
        )
        assert ctx.input_versions == {}
        assert ctx.run_metadata == {}

    def test_default_dicts_are_independent(self) -> None:
        """Each instance gets its own dict, not a shared mutable default."""
        ctx1 = AssetContext(
            partition="p1",
            asset_name="a",
            compute_key="k",
            artifact_path="/p",
            necessity="active",
        )
        ctx2 = AssetContext(
            partition="p2",
            asset_name="b",
            compute_key="k",
            artifact_path="/p",
            necessity="active",
        )
        ctx1.input_versions["x"] = "1"
        assert "x" not in ctx2.input_versions


class TestDevContext:
    def test_creation_with_defaults(self) -> None:
        ctx = DevContext(partition="2026-03-01")
        assert ctx.partition == "2026-03-01"
        assert ctx.asset_name == "dev"
        assert ctx.compute_key == "dev"
        assert ctx.artifact_path == "/tmp/dev"
        assert ctx.necessity == "active"
        assert ctx.input_versions == {}
        assert ctx.run_metadata == {}

    def test_creation_with_overrides(self) -> None:
        ctx = DevContext(
            partition="2026-03-01",
            asset_name="my_asset",
            compute_key="test_key",
        )
        assert ctx.asset_name == "my_asset"
        assert ctx.compute_key == "test_key"
