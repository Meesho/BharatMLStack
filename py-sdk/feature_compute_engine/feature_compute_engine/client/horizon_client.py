"""
HTTP client for the horizon Feature Computation Engine API.

Used by:
- Airflow AssetExecutionOperator (get plan, report results)
- CLI tools (register assets, propose plans)
- DAG generator (list assets/notebooks)
"""
from __future__ import annotations

import json
import logging
import time
from dataclasses import dataclass
from typing import Any, Dict, List, Optional
from urllib.parse import urljoin

from ..types.asset_spec import AssetSpec
from ..types.execution_plan import NotebookExecutionPlan

logger = logging.getLogger(__name__)


@dataclass
class HorizonClientConfig:
    base_url: str
    api_prefix: str = "/api/v1/fce"
    api_key: Optional[str] = None
    timeout_seconds: int = 30
    max_retries: int = 3
    retry_delay_seconds: float = 1.0


class HorizonClientError(Exception):
    """Raised when horizon API returns an error."""

    def __init__(self, status_code: int, message: str, endpoint: str):
        self.status_code = status_code
        self.endpoint = endpoint
        super().__init__(
            f"Horizon API error ({status_code}) on {endpoint}: {message}"
        )


class HorizonClient:
    """Client for horizon Feature Computation Engine APIs."""

    def __init__(self, config: HorizonClientConfig):
        self._config = config
        self._session = None

    @property
    def session(self):
        if self._session is None:
            import requests

            self._session = requests.Session()
            self._session.headers.update(
                {
                    "Content-Type": "application/json",
                    "Accept": "application/json",
                }
            )
            if self._config.api_key:
                self._session.headers["Authorization"] = (
                    f"Bearer {self._config.api_key}"
                )
        return self._session

    def _url(self, path: str) -> str:
        return urljoin(
            self._config.base_url, f"{self._config.api_prefix}{path}"
        )

    def _request(self, method: str, path: str, **kwargs) -> Dict[str, Any]:
        """Make an HTTP request with retry logic for transient failures."""
        url = self._url(path)
        kwargs.setdefault("timeout", self._config.timeout_seconds)

        last_error: Optional[Exception] = None
        for attempt in range(1, self._config.max_retries + 1):
            try:
                response = self.session.request(method, url, **kwargs)
                if response.status_code >= 400:
                    raise HorizonClientError(
                        status_code=response.status_code,
                        message=response.text,
                        endpoint=f"{method} {path}",
                    )
                return response.json() if response.content else {}
            except HorizonClientError:
                raise
            except Exception as e:
                last_error = e
                if attempt < self._config.max_retries:
                    delay = self._config.retry_delay_seconds * attempt
                    logger.warning(
                        "Retry %d/%d for %s: %s",
                        attempt,
                        self._config.max_retries,
                        path,
                        e,
                    )
                    time.sleep(delay)

        raise RuntimeError(
            f"Failed after {self._config.max_retries} retries: {last_error}"
        )

    # === Execution Plan ===

    def get_execution_plan(
        self,
        notebook: str,
        trigger_type: str,
        partition: str,
    ) -> NotebookExecutionPlan:
        """
        Get execution plan for a notebook invocation.
        Called by AssetExecutionOperator before launching Databricks.
        """
        data = self._request(
            "POST",
            "/execution-plan",
            json={
                "notebook": notebook,
                "trigger_type": trigger_type,
                "partition": partition,
            },
        )
        return NotebookExecutionPlan.from_json(json.dumps(data))

    # === Asset Status Reporting ===

    def report_asset_ready(
        self,
        asset_name: str,
        partition: str,
        compute_key: str,
        artifact_path: str,
        delta_version: int = 0,
    ) -> List[Dict[str, Any]]:
        """
        Report that an asset completed successfully.
        Returns trigger actions for downstream assets.
        """
        data = self._request(
            "POST",
            "/assets/ready",
            json={
                "asset_name": asset_name,
                "partition": partition,
                "compute_key": compute_key,
                "artifact_path": artifact_path,
                "delta_version": delta_version,
            },
        )
        return data.get("trigger_actions", [])

    def report_asset_failed(
        self,
        asset_name: str,
        partition: str,
        error: str,
    ) -> None:
        """Report that an asset execution failed."""
        self._request(
            "POST",
            "/assets/failed",
            json={
                "asset_name": asset_name,
                "partition": partition,
                "error": error,
            },
        )

    # === Asset Registration ===

    def register_assets(self, specs: List[AssetSpec]) -> Dict[str, Any]:
        """Register asset specs with horizon."""
        return self._request(
            "POST",
            "/assets/register",
            json={"assets": [s.to_dict() for s in specs]},
        )

    # === Plan Stage ===

    def propose_plan(
        self, specs: List[AssetSpec], dry_run: bool = True
    ) -> Dict[str, Any]:
        """Propose changes and get impact analysis."""
        return self._request(
            "POST",
            "/plan",
            json={
                "assets": [s.to_dict() for s in specs],
                "dry_run": dry_run,
            },
        )

    def apply_plan(self, plan_id: str) -> Dict[str, Any]:
        """Apply a previously proposed plan."""
        return self._request("POST", f"/plan/{plan_id}/apply")

    # === Necessity & Serving ===

    def get_necessity(
        self, asset_name: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get necessity state for one or all assets."""
        if asset_name:
            return self._request("GET", f"/necessity/{asset_name}")
        return self._request("GET", "/necessity")

    def set_serving_override(
        self,
        asset_name: str,
        serving: bool,
        reason: str,
        updated_by: str,
    ) -> None:
        """Set runtime serving override for an asset."""
        self._request(
            "POST",
            "/serving/override",
            json={
                "asset_name": asset_name,
                "serving": serving,
                "reason": reason,
                "updated_by": updated_by,
            },
        )

    # === Lineage ===

    def get_lineage(
        self, asset_name: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get full lineage graph or upstream/downstream for a specific asset."""
        if asset_name:
            return self._request(
                "GET", f"/lineage/{asset_name}/upstream"
            )
        return self._request("GET", "/lineage")

    # === Dataset Readiness ===

    def mark_dataset_ready(
        self,
        dataset_name: str,
        partition: str,
        delta_version: int,
    ) -> List[Dict[str, Any]]:
        """
        Mark an external dataset partition as ready.
        Used by upstream pipelines to trigger feature computation.
        Returns trigger actions.
        """
        data = self._request(
            "POST",
            "/datasets/ready",
            json={
                "dataset_name": dataset_name,
                "partition": partition,
                "delta_version": delta_version,
            },
        )
        return data.get("trigger_actions", [])

    # === Notebook/Asset Discovery ===

    def list_notebooks(self) -> List[Dict[str, Any]]:
        """List all registered notebooks with their assets."""
        data = self._request("GET", "/assets")
        return data.get("notebooks", [])
