"""
NotebookRuntime — the execution engine that runs inside Databricks notebooks.

This is the entry point called by the Airflow-launched Databricks job.
It reads the execution plan from notebook parameters, discovers registered
assets, and executes the matching subset.

Usage (production — called automatically)::

    if __name__ == "__main__":
        NotebookRuntime.run()

Usage (interactive development)::

    ctx = DevContext(partition="2026-03-01")
    result = user_spend(ctx, "2026-03-01", my_orders_df)
"""
from __future__ import annotations

import json
import logging
import time
from typing import Any, Dict, List, Optional

from ..types.execution_plan import NotebookExecutionPlan, AssetExecutionPlan
from ..decorators.registry import AssetRegistry
from .context import AssetContext
from .input_reader import InputReader
from .output_writer import OutputWriter
from .dq_checker import DQChecker
from .result import AssetResult, NotebookRunResult

logger = logging.getLogger(__name__)


class NotebookRuntime:
    """Orchestrates asset execution within a single notebook invocation."""

    @staticmethod
    def run(
        plan_json: Optional[str] = None,
        engine: str = "spark",
    ) -> NotebookRunResult:
        """
        Main entry point. Called by the Databricks notebook.

        Args:
            plan_json: Execution plan as JSON. If None, reads from
                       Databricks notebook widget/parameter ``execution_plan``.
            engine: ``"spark"`` or ``"polars"``

        Returns:
            NotebookRunResult with per-asset outcomes.
        """
        if plan_json is None:
            plan_json = NotebookRuntime._read_dbutils_param("execution_plan")
        plan = NotebookExecutionPlan.from_json(plan_json)

        logger.info(
            "NotebookRuntime starting: notebook=%s, trigger=%s, "
            "partition=%s, assets=%d (%d to execute)",
            plan.notebook,
            plan.trigger_type,
            plan.partition,
            len(plan.assets),
            len(plan.assets_to_execute),
        )

        if not plan.has_work:
            logger.info("No assets to execute — all skipped")
            return NotebookRunResult(
                notebook=plan.notebook,
                trigger_type=plan.trigger_type,
                partition=plan.partition,
                results=[
                    AssetResult(
                        asset_name=a.asset_name,
                        action=f"skipped_{a.action.replace('skip_', '')}",
                        reason=a.reason,
                    )
                    for a in plan.assets
                ],
            )

        reader = InputReader(engine=engine)
        writer = OutputWriter(engine=engine)
        checker = DQChecker()

        shared_dfs = reader.read_shared(plan.shared_inputs)

        registry = AssetRegistry.get_all()

        failed_assets: set = set()
        results: List[AssetResult] = []

        for asset_plan in plan.assets:
            result = NotebookRuntime._execute_single_asset(
                asset_plan=asset_plan,
                plan=plan,
                registry=registry,
                reader=reader,
                writer=writer,
                checker=checker,
                shared_dfs=shared_dfs,
                failed_assets=failed_assets,
                engine=engine,
            )
            results.append(result)

            if result.failed:
                failed_assets.add(asset_plan.asset_name)

        run_result = NotebookRunResult(
            notebook=plan.notebook,
            trigger_type=plan.trigger_type,
            partition=plan.partition,
            results=results,
        )

        NotebookRuntime._write_results(run_result)

        logger.info(
            "NotebookRuntime complete: executed=%d, failed=%d, skipped=%d",
            run_result.executed_count,
            run_result.failed_count,
            run_result.skipped_count,
        )

        return run_result

    @staticmethod
    def _execute_single_asset(
        asset_plan: AssetExecutionPlan,
        plan: NotebookExecutionPlan,
        registry: Dict,
        reader: InputReader,
        writer: OutputWriter,
        checker: DQChecker,
        shared_dfs: Dict[str, Any],
        failed_assets: set,
        engine: str,
    ) -> AssetResult:
        """Execute or skip a single asset based on its plan."""

        if not asset_plan.should_execute:
            logger.info("Skipping %s: %s", asset_plan.asset_name, asset_plan.action)
            return AssetResult(
                asset_name=asset_plan.asset_name,
                action=f"skipped_{asset_plan.action.replace('skip_', '')}",
                reason=asset_plan.reason,
            )

        spec = registry.get(asset_plan.asset_name)
        if spec:
            upstream_in_notebook = [
                inp.name for inp in spec.inputs if inp.name in registry
            ]
            failed_upstreams = [u for u in upstream_in_notebook if u in failed_assets]
            if failed_upstreams:
                reason = f"Upstream failed: {failed_upstreams}"
                logger.warning("Skipping %s: %s", asset_plan.asset_name, reason)
                return AssetResult(
                    asset_name=asset_plan.asset_name,
                    action="failed",
                    error=reason,
                )

        logger.info("Executing %s", asset_plan.asset_name)
        start_time = time.monotonic()

        try:
            if spec is None or spec._compute_fn is None:
                raise RuntimeError(
                    f"Asset '{asset_plan.asset_name}' not found in registry. "
                    f"Is it decorated with @asset in this notebook?"
                )
            compute_fn = spec._compute_fn

            input_dfs: Dict[str, Any] = {}
            for inp in spec.inputs:
                if inp.name in shared_dfs:
                    input_dfs[inp.name] = shared_dfs[inp.name]
                elif inp.name in asset_plan.input_bindings:
                    input_dfs[inp.name] = reader.read(
                        inp.name,
                        asset_plan.input_bindings[inp.name],
                        plan.partition,
                    )
                else:
                    raise RuntimeError(
                        f"Input '{inp.name}' for asset '{asset_plan.asset_name}' "
                        f"has no binding in execution plan and is not in shared inputs"
                    )

            ctx = AssetContext(
                partition=plan.partition,
                asset_name=asset_plan.asset_name,
                compute_key=asset_plan.compute_key or "",
                artifact_path=asset_plan.artifact_path or "",
                necessity=asset_plan.necessity,
                input_versions=asset_plan.input_bindings,
            )

            ordered_dfs = [input_dfs[inp.name] for inp in spec.inputs]
            output_df = compute_fn(ctx, plan.partition, *ordered_dfs)

            dq_results = checker.run_checks(output_df, asset_plan.checks, engine)
            if not checker.all_passed(dq_results):
                failed_checks = [c for c, passed in dq_results.items() if not passed]
                raise RuntimeError(f"DQ checks failed: {failed_checks}")

            if asset_plan.artifact_path:
                writer.write_artifact(output_df, asset_plan.artifact_path)

            if asset_plan.should_publish:
                serving_path = asset_plan.artifact_path.replace(
                    "/_artifacts/", "/serving/"
                )
                writer.publish_to_serving(output_df, serving_path)

            reader.inject_output(asset_plan.asset_name, output_df)

            duration = time.monotonic() - start_time
            row_count = output_df.count() if engine == "spark" else len(output_df)

            return AssetResult(
                asset_name=asset_plan.asset_name,
                action="executed",
                compute_key=asset_plan.compute_key,
                artifact_path=asset_plan.artifact_path,
                row_count=row_count,
                duration_seconds=round(duration, 2),
                dq_results=dq_results,
            )

        except Exception as e:
            duration = time.monotonic() - start_time
            logger.error(
                "Asset %s FAILED: %s", asset_plan.asset_name, e, exc_info=True
            )
            return AssetResult(
                asset_name=asset_plan.asset_name,
                action="failed",
                compute_key=asset_plan.compute_key,
                error=str(e),
                duration_seconds=round(duration, 2),
            )

    @staticmethod
    def _read_dbutils_param(key: str) -> str:
        """Read a parameter from Databricks notebook widgets."""
        try:
            from pyspark.dbutils import DBUtils  # type: ignore[import-untyped]
            from pyspark.sql import SparkSession

            spark = SparkSession.getActiveSession()
            dbutils = DBUtils(spark)
            return dbutils.widgets.get(key)
        except Exception:
            raise RuntimeError(
                f"Could not read notebook parameter '{key}'. "
                f"Are you running inside Databricks with the parameter set?"
            )

    @staticmethod
    def _write_results(result: NotebookRunResult) -> None:
        """Write results as JSON for the Airflow operator to read."""
        try:
            from pyspark.dbutils import DBUtils  # type: ignore[import-untyped]
            from pyspark.sql import SparkSession

            spark = SparkSession.getActiveSession()
            dbutils = DBUtils(spark)
            dbutils.notebook.exit(json.dumps(result.to_dict()))
        except Exception:
            logger.info("Run result: %s", json.dumps(result.to_dict(), indent=2))
