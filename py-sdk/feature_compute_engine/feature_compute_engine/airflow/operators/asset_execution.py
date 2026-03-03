"""
AssetExecutionOperator — Airflow custom operator that bridges horizon and Databricks.

This is the central piece of the integration:
1. Asks horizon for execution plan (what assets to run)
2. If nothing to execute -> short circuit
3. Launches Databricks notebook with the plan
4. Waits for completion
5. Reports results to horizon (asset ready/failed)
6. Triggers downstream DAGs based on horizon's trigger evaluation

Usage in an Airflow DAG:
    AssetExecutionOperator(
        task_id="execute",
        notebook="user_features",
        trigger_type="schedule_3h",
        horizon_config=HorizonClientConfig(base_url="http://horizon:8080"),
        databricks_conn_id="databricks_default",
        notebook_path="/Repos/prod/features/user_features",
    )
"""
from __future__ import annotations

import json
import logging
from typing import Any, Dict, Optional

from airflow.exceptions import AirflowSkipException
from airflow.models import BaseOperator
from airflow.providers.databricks.hooks.databricks import DatabricksHook

from feature_compute_engine.client.airflow_trigger import (
    AirflowTrigger,
    AirflowTriggerConfig,
)
from feature_compute_engine.client.horizon_client import (
    HorizonClient,
    HorizonClientConfig,
)

logger = logging.getLogger(__name__)


class AssetExecutionOperator(BaseOperator):
    """
    Executes feature computation assets in a Databricks notebook.

    Lifecycle:
    1. Get execution plan from horizon
    2. Short-circuit if nothing to execute
    3. Submit Databricks notebook run
    4. Wait for completion
    5. Parse results
    6. Report each asset result to horizon
    7. Trigger downstream DAGs
    """

    template_fields = ("partition",)

    def __init__(
        self,
        *,
        notebook: str,
        trigger_type: str,
        horizon_config: Optional[HorizonClientConfig] = None,
        horizon_conn_id: Optional[str] = "horizon_default",
        databricks_conn_id: str = "databricks_default",
        notebook_path: Optional[str] = None,
        cluster_config: Optional[Dict[str, Any]] = None,
        partition: str = "{{ ds }}",
        pool: str = "databricks_pool",
        airflow_trigger_config: Optional[AirflowTriggerConfig] = None,
        **kwargs,
    ):
        super().__init__(pool=pool, **kwargs)
        self.notebook = notebook
        self.trigger_type = trigger_type
        self.horizon_config = horizon_config
        self.horizon_conn_id = horizon_conn_id
        self.databricks_conn_id = databricks_conn_id
        self.notebook_path = notebook_path or f"/Repos/prod/features/{notebook}"
        self.cluster_config = cluster_config or {}
        self.partition = partition
        self.airflow_trigger_config = airflow_trigger_config

    def execute(self, context: Dict[str, Any]) -> Dict[str, Any]:
        """Main execution method called by Airflow."""
        partition = self.partition

        horizon = self._get_horizon_client()
        plan = horizon.get_execution_plan(
            notebook=self.notebook,
            trigger_type=self.trigger_type,
            partition=partition,
        )

        logger.info(
            "Execution plan: notebook=%s, trigger=%s, partition=%s, "
            "total_assets=%d, to_execute=%d",
            plan.notebook,
            plan.trigger_type,
            plan.partition,
            len(plan.assets),
            len(plan.assets_to_execute),
        )

        if not plan.has_work:
            logger.info(
                "All assets skipped (cached or unnecessary). Short-circuiting."
            )
            raise AirflowSkipException(
                "All assets skipped — no Databricks job needed"
            )

        plan_json = plan.to_json()
        run_id = self._submit_databricks_run(plan_json, partition)
        logger.info("Databricks run submitted: run_id=%s", run_id)

        hook = DatabricksHook(databricks_conn_id=self.databricks_conn_id)
        hook.wait_for_run(run_id)

        run_output = hook.get_run_output(run_id)
        notebook_output = run_output.get("notebook_output", {})
        result_json = notebook_output.get("result", "{}")

        try:
            results = json.loads(result_json)
        except json.JSONDecodeError:
            logger.error(
                "Failed to parse notebook output: %s", result_json[:500]
            )
            results = {"results": []}

        all_trigger_actions = []
        for asset_result in results.get("results", []):
            asset_name = asset_result["asset_name"]
            action = asset_result["action"]

            if action == "executed":
                trigger_actions = horizon.report_asset_ready(
                    asset_name=asset_name,
                    partition=partition,
                    compute_key=asset_result.get("compute_key", ""),
                    artifact_path=asset_result.get("artifact_path", ""),
                )
                all_trigger_actions.extend(trigger_actions)
                logger.info(
                    "Asset %s: READY (triggers: %d)",
                    asset_name,
                    len(trigger_actions),
                )

            elif action == "failed":
                horizon.report_asset_failed(
                    asset_name=asset_name,
                    partition=partition,
                    error=asset_result.get("error", "Unknown error"),
                )
                logger.warning(
                    "Asset %s: FAILED — %s",
                    asset_name,
                    asset_result.get("error"),
                )

            else:
                logger.info("Asset %s: %s", asset_name, action)

        if all_trigger_actions:
            logger.info(
                "Triggering %d downstream DAGs", len(all_trigger_actions)
            )
            trigger = self._get_airflow_trigger()
            trigger.trigger_from_actions(all_trigger_actions)

        return results

    def _submit_databricks_run(
        self, plan_json: str, partition: str
    ) -> str:
        """Submit a Databricks notebook run with the execution plan."""
        hook = DatabricksHook(databricks_conn_id=self.databricks_conn_id)

        run_config: Dict[str, Any] = {
            "notebook_task": {
                "notebook_path": self.notebook_path,
                "base_parameters": {
                    "execution_plan": plan_json,
                    "partition": partition,
                },
            },
        }

        if self.cluster_config:
            if "existing_cluster_id" in self.cluster_config:
                run_config["existing_cluster_id"] = self.cluster_config[
                    "existing_cluster_id"
                ]
            elif "new_cluster" in self.cluster_config:
                run_config["new_cluster"] = self.cluster_config["new_cluster"]

        run_id = hook.submit_run(run_config)
        return str(run_id)

    def _get_horizon_client(self) -> HorizonClient:
        """Build horizon client from config or Airflow connection."""
        if self.horizon_config:
            return HorizonClient(self.horizon_config)

        from airflow.hooks.base import BaseHook

        conn = BaseHook.get_connection(self.horizon_conn_id)
        config = HorizonClientConfig(
            base_url=(
                f"{conn.schema or 'http'}://{conn.host}:{conn.port or 8080}"
            ),
            api_key=conn.password,
        )
        return HorizonClient(config)

    def _get_airflow_trigger(self) -> AirflowTrigger:
        """Build Airflow trigger client."""
        if self.airflow_trigger_config:
            return AirflowTrigger(self.airflow_trigger_config)
        return AirflowTrigger(
            AirflowTriggerConfig(base_url="http://localhost:8080")
        )
