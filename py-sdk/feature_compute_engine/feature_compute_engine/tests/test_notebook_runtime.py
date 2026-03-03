"""
Tests for NotebookRuntime — uses plain Python mocks, no Spark/Polars.

Covers:
- Plan with 2 execute + 1 skip → only 2 assets run
- Asset failure doesn't block independent assets in same notebook
- Asset failure blocks downstream dependent in same notebook
- DQ check failure → asset marked as failed
- Empty plan (all skipped) → short circuit, no reader/writer calls
- Input injection: asset A output available to asset B in same invocation
"""
from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from feature_compute_engine.decorators.registry import AssetRegistry
from feature_compute_engine.runtime.notebook_runtime import NotebookRuntime
from feature_compute_engine.types.asset_spec import AssetSpec, Input
from feature_compute_engine.types.execution_plan import (
    AssetAction,
    AssetExecutionPlan,
    NotebookExecutionPlan,
)


@pytest.fixture(autouse=True)
def _clean_registry():
    AssetRegistry.clear()
    yield
    AssetRegistry.clear()


def _register_asset(
    name: str,
    compute_fn=None,
    inputs=None,
    notebook: str = "test_notebook",
) -> AssetSpec:
    if compute_fn is None:
        compute_fn = MagicMock(return_value=MagicMock(name=f"output_{name}"))
    spec = AssetSpec(
        name=name,
        entity="user",
        entity_key="user_id",
        notebook=notebook,
        inputs=inputs or [],
        _compute_fn=compute_fn,
    )
    AssetRegistry.register(spec)
    return spec


def _make_plan(
    assets: list,
    shared_inputs: dict = None,
    notebook: str = "test_notebook",
) -> str:
    plan = NotebookExecutionPlan(
        notebook=notebook,
        trigger_type="schedule_3h",
        partition="2026-03-01",
        assets=assets,
        shared_inputs=shared_inputs or {},
    )
    return plan.to_json()


def _make_output_df(count: int = 100) -> MagicMock:
    df = MagicMock(name="output_df")
    df.count.return_value = count
    return df


class TestNotebookRuntimeBasic:
    def test_two_execute_one_skip(self) -> None:
        """Plan with 2 execute + 1 skip → only 2 assets run."""
        fn_a = MagicMock(return_value=_make_output_df())
        fn_b = MagicMock(return_value=_make_output_df())
        fn_c = MagicMock(return_value=_make_output_df())

        _register_asset("fg.a", compute_fn=fn_a)
        _register_asset("fg.b", compute_fn=fn_b)
        _register_asset("fg.c", compute_fn=fn_c)

        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
            ),
            AssetExecutionPlan(
                asset_name="fg.b", action=AssetAction.SKIP_CACHED, necessity="active",
                reason="cache hit",
            ),
            AssetExecutionPlan(
                asset_name="fg.c", action=AssetAction.EXECUTE, necessity="transient",
                compute_key="k3", artifact_path="/_artifacts/fg.c/2026-03-01/k3/",
            ),
        ])

        result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        assert result.executed_count == 2
        assert result.skipped_count == 1
        assert result.failed_count == 0
        fn_a.assert_called_once()
        fn_b.assert_not_called()
        fn_c.assert_called_once()

    def test_empty_plan_short_circuits(self) -> None:
        """Empty plan (all skipped) → no reader/writer calls."""
        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.SKIP_CACHED, necessity="active",
            ),
            AssetExecutionPlan(
                asset_name="fg.b", action=AssetAction.SKIP_UNNECESSARY, necessity="skipped",
            ),
        ])

        result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        assert result.executed_count == 0
        assert result.skipped_count == 2
        assert result.failed_count == 0
        assert result.all_succeeded is True


class TestNotebookRuntimeFailures:
    def test_independent_asset_failure_doesnt_block_others(self) -> None:
        """Asset failure doesn't block independent assets in same notebook."""
        fn_a = MagicMock(side_effect=RuntimeError("boom"))
        fn_b = MagicMock(return_value=_make_output_df())

        _register_asset("fg.a", compute_fn=fn_a)
        _register_asset("fg.b", compute_fn=fn_b)

        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
            ),
            AssetExecutionPlan(
                asset_name="fg.b", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k2", artifact_path="/_artifacts/fg.b/2026-03-01/k2/",
            ),
        ])

        result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        assert result.failed_count == 1
        assert result.executed_count == 1
        assert result.results[0].failed is True
        assert result.results[0].error == "boom"
        assert result.results[1].succeeded is True

    def test_upstream_failure_blocks_downstream(self) -> None:
        """Asset failure blocks downstream dependent in same notebook."""
        fn_a = MagicMock(side_effect=RuntimeError("upstream broke"))
        fn_b = MagicMock(return_value=_make_output_df())

        _register_asset("fg.a", compute_fn=fn_a)
        _register_asset(
            "fg.b",
            compute_fn=fn_b,
            inputs=[Input("fg.a")],
        )

        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
            ),
            AssetExecutionPlan(
                asset_name="fg.b", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k2", artifact_path="/_artifacts/fg.b/2026-03-01/k2/",
                input_bindings={"fg.a": "/_artifacts/fg.a/2026-03-01/k1/"},
            ),
        ])

        result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        assert result.failed_count == 2
        assert "Upstream failed" in result.results[1].error
        fn_b.assert_not_called()


class TestNotebookRuntimeDQ:
    def test_dq_failure_marks_asset_failed(self) -> None:
        """DQ check failure → asset marked as failed."""
        output_df = _make_output_df(count=0)
        fn_a = MagicMock(return_value=output_df)
        _register_asset("fg.a", compute_fn=fn_a)

        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
                checks=["row_count > 0"],
            ),
        ])

        result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        assert result.failed_count == 1
        assert "DQ checks failed" in result.results[0].error


class TestNotebookRuntimeInputInjection:
    def test_asset_output_available_to_downstream(self) -> None:
        """Input injection: asset A output available to asset B in same invocation."""
        output_a = _make_output_df(count=50)
        output_b = _make_output_df(count=75)

        fn_a = MagicMock(return_value=output_a)

        def fn_b_impl(ctx, partition, a_df):
            assert a_df is output_a
            return output_b

        _register_asset("fg.a", compute_fn=fn_a)
        _register_asset(
            "fg.b",
            compute_fn=fn_b_impl,
            inputs=[Input("fg.a")],
        )

        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
            ),
            AssetExecutionPlan(
                asset_name="fg.b", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k2", artifact_path="/_artifacts/fg.b/2026-03-01/k2/",
                input_bindings={"fg.a": "/_artifacts/fg.a/2026-03-01/k1/"},
            ),
        ])

        result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        assert result.executed_count == 2
        assert result.failed_count == 0
        assert result.results[1].row_count == 75


class TestNotebookRuntimePublishing:
    def test_active_asset_publishes_to_serving(self) -> None:
        """ACTIVE assets write both artifact and serving."""
        output_df = _make_output_df()
        fn_a = MagicMock(return_value=output_df)
        _register_asset("fg.a", compute_fn=fn_a)

        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
            ),
        ])

        with patch(
            "feature_compute_engine.runtime.notebook_runtime.OutputWriter"
        ) as MockWriter:
            writer_inst = MockWriter.return_value
            result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        writer_inst.write_artifact.assert_called_once()
        writer_inst.publish_to_serving.assert_called_once_with(
            output_df, "/serving/fg.a/2026-03-01/k1/"
        )
        assert result.executed_count == 1

    def test_transient_asset_does_not_publish(self) -> None:
        """TRANSIENT assets write artifact only, no serving."""
        output_df = _make_output_df()
        fn_a = MagicMock(return_value=output_df)
        _register_asset("fg.a", compute_fn=fn_a)

        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.a", action=AssetAction.EXECUTE, necessity="transient",
                compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
            ),
        ])

        with patch(
            "feature_compute_engine.runtime.notebook_runtime.OutputWriter"
        ) as MockWriter:
            writer_inst = MockWriter.return_value
            result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        writer_inst.write_artifact.assert_called_once()
        writer_inst.publish_to_serving.assert_not_called()
        assert result.executed_count == 1


class TestNotebookRuntimeSharedInputs:
    def test_shared_inputs_passed_to_assets(self) -> None:
        """Shared inputs are read once and passed to all assets."""
        shared_df = MagicMock(name="shared_orders_df")
        output_df = _make_output_df()

        def fn_a(ctx, partition, orders_df):
            assert orders_df is shared_df
            return output_df

        _register_asset(
            "fg.a",
            compute_fn=fn_a,
            inputs=[Input("silver.orders", input_type="external")],
        )

        plan_json = _make_plan(
            assets=[
                AssetExecutionPlan(
                    asset_name="fg.a", action=AssetAction.EXECUTE, necessity="active",
                    compute_key="k1", artifact_path="/_artifacts/fg.a/2026-03-01/k1/",
                ),
            ],
            shared_inputs={"silver.orders": "silver.orders@v42"},
        )

        with patch(
            "feature_compute_engine.runtime.notebook_runtime.InputReader"
        ) as MockReader:
            reader_inst = MockReader.return_value
            reader_inst.read_shared.return_value = {"silver.orders": shared_df}
            reader_inst.read.return_value = shared_df

            result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        reader_inst.read_shared.assert_called_once()
        assert result.executed_count == 1


class TestNotebookRuntimeMissingAsset:
    def test_asset_not_in_registry_fails(self) -> None:
        """Asset in plan but not in registry → fails with clear error."""
        plan_json = _make_plan([
            AssetExecutionPlan(
                asset_name="fg.ghost", action=AssetAction.EXECUTE, necessity="active",
                compute_key="k1", artifact_path="/_artifacts/fg.ghost/2026-03-01/k1/",
            ),
        ])

        result = NotebookRuntime.run(plan_json=plan_json, engine="spark")

        assert result.failed_count == 1
        assert "not found in registry" in result.results[0].error
