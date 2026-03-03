"""
DatasetReadySensor — waits for an external dataset partition to be ready in horizon.

Used as a fallback for external data sources that don't actively notify horizon.
Prefer push-based readiness (calling horizon's /datasets/ready endpoint) when possible.

Usage:
    DatasetReadySensor(
        task_id="wait_for_orders",
        dataset_name="silver.orders",
        partition="{{ ds }}",
        horizon_conn_id="horizon_default",
        poke_interval=300,  # check every 5 minutes
        timeout=3600,       # give up after 1 hour
    )
"""
from __future__ import annotations

import logging
from typing import Any, Dict, Optional

from airflow.sensors.base import BaseSensorOperator

from feature_compute_engine.client.horizon_client import (
    HorizonClient,
    HorizonClientConfig,
)

logger = logging.getLogger(__name__)


class DatasetReadySensor(BaseSensorOperator):
    """Waits for a dataset partition to be marked ready in horizon."""

    template_fields = ("partition",)

    def __init__(
        self,
        *,
        dataset_name: str,
        partition: str = "{{ ds }}",
        horizon_config: Optional[HorizonClientConfig] = None,
        horizon_conn_id: str = "horizon_default",
        **kwargs,
    ):
        super().__init__(**kwargs)
        self.dataset_name = dataset_name
        self.partition = partition
        self.horizon_config = horizon_config
        self.horizon_conn_id = horizon_conn_id

    def poke(self, context: Dict[str, Any]) -> bool:
        """Check if the dataset partition is ready."""
        horizon = self._get_horizon_client()
        try:
            result = horizon._request(
                "GET",
                f"/datasets/{self.dataset_name}/partitions",
            )
            partitions = result.get("partitions", [])
            for p in partitions:
                if (
                    p.get("partition_key") == self.partition
                    and p.get("is_ready")
                ):
                    logger.info(
                        "Dataset %s partition %s is READY",
                        self.dataset_name,
                        self.partition,
                    )
                    return True
            logger.info(
                "Dataset %s partition %s not yet ready",
                self.dataset_name,
                self.partition,
            )
            return False
        except Exception as e:
            logger.warning("Error checking readiness: %s", e)
            return False

    def _get_horizon_client(self) -> HorizonClient:
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
