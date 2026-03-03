"""
Triggers Airflow DAGs via the Airflow REST API.
Called by horizon (indirectly, through the operator) to cascade triggers.
"""
from __future__ import annotations

import logging
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

logger = logging.getLogger(__name__)


@dataclass
class AirflowTriggerConfig:
    base_url: str
    username: Optional[str] = None
    password: Optional[str] = None
    timeout_seconds: int = 15


class AirflowTrigger:
    """Triggers Airflow DAGs via the stable REST API."""

    def __init__(self, config: AirflowTriggerConfig):
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
            if self._config.username and self._config.password:
                self._session.auth = (
                    self._config.username,
                    self._config.password,
                )
        return self._session

    def trigger_dag(
        self,
        dag_id: str,
        conf: Optional[Dict[str, Any]] = None,
        logical_date: Optional[str] = None,
    ) -> str:
        """
        Trigger an Airflow DAG run via REST API.

        Args:
            dag_id: The Airflow DAG to trigger (e.g. "fce__user_features__upstream")
            conf: Configuration to pass to the DAG run
            logical_date: Optional execution date

        Returns:
            The DAG run ID
        """
        url = f"{self._config.base_url}/api/v1/dags/{dag_id}/dagRuns"
        payload: Dict[str, Any] = {}
        if conf:
            payload["conf"] = conf
        if logical_date:
            payload["logical_date"] = logical_date

        response = self.session.post(
            url, json=payload, timeout=self._config.timeout_seconds
        )
        response.raise_for_status()

        data = response.json()
        run_id = data.get("dag_run_id", "unknown")
        logger.info("Triggered DAG '%s' -> run_id=%s", dag_id, run_id)
        return run_id

    def trigger_from_actions(
        self, trigger_actions: List[Dict[str, Any]]
    ) -> List[str]:
        """
        Process trigger actions returned by horizon and trigger the corresponding DAGs.

        Each trigger action has: {notebook, trigger_type, partition, asset_name}
        Maps to DAG ID convention: fce__{notebook}__{trigger_type}
        """
        run_ids: List[str] = []
        for action in trigger_actions:
            dag_id = f"fce__{action['notebook']}__{action['trigger_type']}"
            conf = {
                "partition": action["partition"],
                "triggered_by": action.get("asset_name", "unknown"),
            }
            try:
                run_id = self.trigger_dag(dag_id, conf=conf)
                run_ids.append(run_id)
            except Exception as e:
                logger.error("Failed to trigger DAG '%s': %s", dag_id, e)
        return run_ids
