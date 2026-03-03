"""Tests for AssetExecutionOperator — the main Airflow integration."""
from __future__ import annotations

import json
import sys
from unittest.mock import MagicMock, patch

import pytest

from feature_compute_engine.types.execution_plan import (
    AssetAction,
    AssetExecutionPlan,
    NotebookExecutionPlan,
)


# Wire up AirflowSkipException as a real exception subclass so pytest.raises works
class _AirflowSkipException(Exception):
    pass


# Mock Airflow modules persistently before importing the operator, since
# Airflow is not installed in the test environment.
_airflow_mocks = {}
for _mod_name in [
    "airflow",
    "airflow.models",
    "airflow.exceptions",
    "airflow.hooks",
    "airflow.hooks.base",
    "airflow.sensors",
    "airflow.sensors.base",
    "airflow.providers",
    "airflow.providers.databricks",
    "airflow.providers.databricks.hooks",
    "airflow.providers.databricks.hooks.databricks",
]:
    _airflow_mocks[_mod_name] = MagicMock()

_airflow_mocks["airflow.exceptions"].AirflowSkipException = _AirflowSkipException

_BaseOperator = type("BaseOperator", (), {"__init__": lambda self, **kw: None})
_airflow_mocks["airflow.models"].BaseOperator = _BaseOperator

_BaseSensorOperator = type(
    "BaseSensorOperator", (), {"__init__": lambda self, **kw: None}
)
_airflow_mocks["airflow.sensors.base"].BaseSensorOperator = _BaseSensorOperator

sys.modules.update(_airflow_mocks)

# Now safe to import the operator module
import feature_compute_engine.airflow.operators.asset_execution as _asset_exec_mod
from feature_compute_engine.airflow.operators.asset_execution import (
    AssetExecutionOperator,
)


def _make_plan(assets=None, has_work=True) -> NotebookExecutionPlan:
    if assets is None:
        assets = [
            AssetExecutionPlan(
                asset_name="fg.user_spend",
                action=AssetAction.EXECUTE
                if has_work
                else AssetAction.SKIP_CACHED,
                necessity="active",
                compute_key="abc123",
                artifact_path="/_artifacts/fg.user_spend/2025-01-15/abc123/",
            ),
        ]
    return NotebookExecutionPlan(
        notebook="user_features",
        trigger_type="schedule_3h",
        partition="2025-01-15",
        assets=assets,
    )


def _make_notebook_output(results):
    """Build Databricks notebook output payload."""
    return {
        "notebook_output": {
            "result": json.dumps({"results": results}),
        }
    }


class TestAssetExecutionOperatorHappyPath:
    def test_full_lifecycle(self) -> None:
        """Plan -> execute -> report -> trigger downstream."""
        from feature_compute_engine.client.horizon_client import (
            HorizonClientConfig,
        )

        plan = _make_plan()
        mock_horizon = MagicMock()
        mock_horizon.get_execution_plan.return_value = plan
        mock_horizon.report_asset_ready.return_value = [
            {
                "notebook": "product_features",
                "trigger_type": "upstream",
                "partition": "2025-01-15",
                "asset_name": "fg.user_spend",
            }
        ]

        mock_hook = MagicMock()
        mock_hook.submit_run.return_value = 12345
        mock_hook.get_run_output.return_value = _make_notebook_output(
            [
                {
                    "asset_name": "fg.user_spend",
                    "action": "executed",
                    "compute_key": "abc123",
                    "artifact_path": "/_artifacts/fg.user_spend/2025-01-15/abc123/",
                }
            ]
        )

        mock_trigger = MagicMock()
        mock_trigger.trigger_from_actions.return_value = ["run-1"]

        op = AssetExecutionOperator(
            task_id="test",
            notebook="user_features",
            trigger_type="schedule_3h",
            horizon_config=HorizonClientConfig(
                base_url="http://horizon:8080"
            ),
        )
        op._get_horizon_client = MagicMock(return_value=mock_horizon)
        op._get_airflow_trigger = MagicMock(return_value=mock_trigger)

        with patch.object(
            _asset_exec_mod, "DatabricksHook", return_value=mock_hook
        ):
            result = op.execute(context={})

        mock_horizon.get_execution_plan.assert_called_once_with(
            notebook="user_features",
            trigger_type="schedule_3h",
            partition="{{ ds }}",
        )
        mock_hook.submit_run.assert_called_once()
        mock_horizon.report_asset_ready.assert_called_once()
        mock_trigger.trigger_from_actions.assert_called_once()

        assert result["results"][0]["asset_name"] == "fg.user_spend"


class TestAssetExecutionOperatorShortCircuit:
    def test_all_skipped(self) -> None:
        """When all assets are skipped, operator should raise skip exception."""
        from feature_compute_engine.client.horizon_client import (
            HorizonClientConfig,
        )

        plan = _make_plan(has_work=False)
        mock_horizon = MagicMock()
        mock_horizon.get_execution_plan.return_value = plan

        op = AssetExecutionOperator(
            task_id="test",
            notebook="user_features",
            trigger_type="schedule_3h",
            horizon_config=HorizonClientConfig(
                base_url="http://horizon:8080"
            ),
        )
        op._get_horizon_client = MagicMock(return_value=mock_horizon)

        with pytest.raises(_AirflowSkipException):
            op.execute(context={})


class TestAssetExecutionOperatorPartialFailure:
    def test_some_succeed_some_fail(self) -> None:
        """Both successes and failures are reported to horizon."""
        from feature_compute_engine.client.horizon_client import (
            HorizonClientConfig,
        )

        assets = [
            AssetExecutionPlan(
                asset_name="fg.user_spend",
                action=AssetAction.EXECUTE,
                necessity="active",
                compute_key="abc",
            ),
            AssetExecutionPlan(
                asset_name="fg.user_clicks",
                action=AssetAction.EXECUTE,
                necessity="active",
                compute_key="def",
            ),
        ]
        plan = _make_plan(assets=assets)

        mock_horizon = MagicMock()
        mock_horizon.get_execution_plan.return_value = plan
        mock_horizon.report_asset_ready.return_value = []

        mock_hook = MagicMock()
        mock_hook.submit_run.return_value = 99
        mock_hook.get_run_output.return_value = _make_notebook_output(
            [
                {
                    "asset_name": "fg.user_spend",
                    "action": "executed",
                    "compute_key": "abc",
                    "artifact_path": "/path/spend",
                },
                {
                    "asset_name": "fg.user_clicks",
                    "action": "failed",
                    "error": "OOM",
                },
            ]
        )

        op = AssetExecutionOperator(
            task_id="test",
            notebook="user_features",
            trigger_type="schedule_3h",
            horizon_config=HorizonClientConfig(
                base_url="http://horizon:8080"
            ),
        )
        op._get_horizon_client = MagicMock(return_value=mock_horizon)
        op._get_airflow_trigger = MagicMock(return_value=MagicMock())

        with patch.object(
            _asset_exec_mod, "DatabricksHook", return_value=mock_hook
        ):
            op.execute(context={})

        mock_horizon.report_asset_ready.assert_called_once_with(
            asset_name="fg.user_spend",
            partition="{{ ds }}",
            compute_key="abc",
            artifact_path="/path/spend",
        )
        mock_horizon.report_asset_failed.assert_called_once_with(
            asset_name="fg.user_clicks",
            partition="{{ ds }}",
            error="OOM",
        )


class TestAssetExecutionOperatorNoTriggers:
    def test_no_downstream_triggers(self) -> None:
        """When horizon returns no trigger actions, no DAGs are triggered."""
        from feature_compute_engine.client.horizon_client import (
            HorizonClientConfig,
        )

        plan = _make_plan()
        mock_horizon = MagicMock()
        mock_horizon.get_execution_plan.return_value = plan
        mock_horizon.report_asset_ready.return_value = []

        mock_hook = MagicMock()
        mock_hook.submit_run.return_value = 42
        mock_hook.get_run_output.return_value = _make_notebook_output(
            [
                {
                    "asset_name": "fg.user_spend",
                    "action": "executed",
                    "compute_key": "abc123",
                    "artifact_path": "/path",
                }
            ]
        )

        mock_trigger = MagicMock()

        op = AssetExecutionOperator(
            task_id="test",
            notebook="user_features",
            trigger_type="schedule_3h",
            horizon_config=HorizonClientConfig(
                base_url="http://horizon:8080"
            ),
        )
        op._get_horizon_client = MagicMock(return_value=mock_horizon)
        op._get_airflow_trigger = MagicMock(return_value=mock_trigger)

        with patch.object(
            _asset_exec_mod, "DatabricksHook", return_value=mock_hook
        ):
            op.execute(context={})

        mock_trigger.trigger_from_actions.assert_not_called()
