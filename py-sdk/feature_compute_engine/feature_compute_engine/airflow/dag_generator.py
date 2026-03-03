"""
DAG Generator — creates Airflow DAG Python files from horizon's asset registry.

Run as a CLI tool or a periodic job:
    python -m feature_compute_engine.airflow.dag_generator \
        --horizon-url http://horizon:8080 \
        --output-dir /opt/airflow/dags/fce/

This generates one DAG file per (notebook x trigger_type) combination:
    fce__user_features__schedule_3h.py
    fce__user_features__schedule_6h.py
    fce__user_features__upstream.py
    fce__product_features__schedule_3h.py
    ...
"""
from __future__ import annotations

import logging
import os
from collections import defaultdict
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple

logger = logging.getLogger(__name__)


DAG_TEMPLATE = '''"""
Auto-generated DAG for {notebook} ({trigger_type}).
DO NOT EDIT — regenerate with: bharatml dag-generate

Assets: {asset_names}
"""
from airflow import DAG
from airflow.utils.dates import days_ago
from datetime import timedelta
from feature_compute_engine.airflow.operators.asset_execution import AssetExecutionOperator

default_args = {{
    "owner": "feature-compute-engine",
    "retries": 2,
    "retry_delay": timedelta(minutes=5),
}}

with DAG(
    dag_id="{dag_id}",
    default_args=default_args,
    schedule_interval={schedule_interval},
    start_date=days_ago(1),
    catchup=False,
    max_active_runs={max_active_runs},
    tags=["fce", "{notebook}", "{trigger_type}"],
) as dag:

    execute = AssetExecutionOperator(
        task_id="execute",
        notebook="{notebook}",
        trigger_type="{trigger_type}",
        notebook_path="{notebook_path}",
        pool="databricks_pool",
    )
'''


SCHEDULE_MAP = {
    "schedule_1h": "0 * * * *",
    "schedule_3h": "0 */3 * * *",
    "schedule_6h": "0 */6 * * *",
    "schedule_12h": "0 */12 * * *",
    "schedule_24h": "0 2 * * *",
    "upstream": None,
}


@dataclass
class NotebookTriggerGroup:
    notebook: str
    trigger_type: str
    asset_names: List[str]
    schedule: Optional[str]
    notebook_path: str


def discover_trigger_groups(
    horizon_base_url: str,
    api_key: Optional[str] = None,
) -> List[NotebookTriggerGroup]:
    """Query horizon for all registered assets and group by notebook x trigger."""
    from feature_compute_engine.client.horizon_client import (
        HorizonClient,
        HorizonClientConfig,
    )

    client = HorizonClient(
        HorizonClientConfig(base_url=horizon_base_url, api_key=api_key)
    )

    groups: Dict[Tuple[str, str], List[str]] = defaultdict(list)
    notebooks_data = client.list_notebooks()

    for nb in notebooks_data:
        notebook_name = nb["notebook"]
        for asset in nb.get("assets", []):
            trigger = asset.get("trigger_type", "upstream")
            groups[(notebook_name, trigger)].append(asset["name"])

    result = []
    for (notebook, trigger), assets in groups.items():
        result.append(
            NotebookTriggerGroup(
                notebook=notebook,
                trigger_type=trigger,
                asset_names=assets,
                schedule=SCHEDULE_MAP.get(trigger),
                notebook_path=f"/Repos/prod/features/{notebook}",
            )
        )
    return result


def generate_dag_file(group: NotebookTriggerGroup) -> str:
    """Generate DAG Python code for a notebook x trigger group."""
    dag_id = f"fce__{group.notebook}__{group.trigger_type}"
    schedule_interval = (
        f'"{group.schedule}"' if group.schedule else "None"
    )
    max_active_runs = 1 if group.trigger_type == "upstream" else 3

    return DAG_TEMPLATE.format(
        notebook=group.notebook,
        trigger_type=group.trigger_type,
        dag_id=dag_id,
        schedule_interval=schedule_interval,
        max_active_runs=max_active_runs,
        asset_names=", ".join(group.asset_names),
        notebook_path=group.notebook_path,
    )


def generate_all(
    horizon_base_url: str,
    output_dir: str,
    api_key: Optional[str] = None,
) -> List[str]:
    """Generate all DAG files and write to output directory."""
    os.makedirs(output_dir, exist_ok=True)

    groups = discover_trigger_groups(horizon_base_url, api_key)
    written_files = []

    for group in groups:
        dag_id = f"fce__{group.notebook}__{group.trigger_type}"
        filename = f"{dag_id}.py"
        filepath = os.path.join(output_dir, filename)

        content = generate_dag_file(group)

        with open(filepath, "w") as f:
            f.write(content)

        written_files.append(filepath)
        logger.info(
            "Generated DAG: %s (%d assets)", filename, len(group.asset_names)
        )

    logger.info(
        "Generated %d DAG files in %s", len(written_files), output_dir
    )
    return written_files


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="Generate Airflow DAGs from horizon"
    )
    parser.add_argument("--horizon-url", required=True)
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--api-key", default=None)
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO)
    generate_all(args.horizon_url, args.output_dir, args.api_key)
