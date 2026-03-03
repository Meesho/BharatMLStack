"""
Apply a manifest to horizon -- INTERNAL NETWORK ONLY.

This is called by Trufflebox (or an internal CI runner) after a PR is merged.
It reads the manifest.json from the merged branch and registers all assets
with horizon, resolves necessity, and generates Airflow DAGs.

THIS CODE NEVER RUNS IN EXTERNAL CI.
"""
from __future__ import annotations

import logging
from typing import Optional

from feature_compute_engine.airflow.dag_generator import generate_all
from feature_compute_engine.client.horizon_client import (
    HorizonClient,
    HorizonClientConfig,
)
from feature_compute_engine.types.asset_spec import AssetSpec

from .manifest import read_manifest

logger = logging.getLogger(__name__)


def apply_manifest(
    manifest_path: str,
    horizon_url: str,
    dag_output_dir: Optional[str] = None,
    horizon_api_key: Optional[str] = None,
) -> dict:
    """
    Register manifest contents with horizon and generate Airflow DAGs.

    Args:
        manifest_path: Path to .bharatml/manifest.json
        horizon_url: Internal horizon URL (e.g. http://horizon.internal:8080)
        dag_output_dir: Where to write Airflow DAG files (None = skip DAG gen)
        horizon_api_key: Optional API key for horizon

    Returns:
        Summary of what was applied
    """
    manifest = read_manifest(manifest_path)
    specs = [AssetSpec.from_dict(a) for a in manifest.get("assets", [])]
    git_sha = manifest.get("git_sha", "unknown")

    logger.info("Applying manifest: %d assets, git_sha=%s", len(specs), git_sha)

    client = HorizonClient(
        HorizonClientConfig(
            base_url=horizon_url,
            api_key=horizon_api_key,
        )
    )

    client.register_assets(specs)
    logger.info("Registered %d assets with horizon", len(specs))

    necessity = client.get_necessity()
    active = 0
    transient = 0
    skipped = 0
    for v in necessity.values():
        if isinstance(v, str):
            if v == "active":
                active += 1
            elif v == "transient":
                transient += 1
            elif v == "skipped":
                skipped += 1
    logger.info(
        "Necessity: %d ACTIVE, %d TRANSIENT, %d SKIPPED", active, transient, skipped
    )

    dag_files: list = []
    if dag_output_dir:
        dag_files = generate_all(horizon_url, dag_output_dir, horizon_api_key)
        logger.info("Generated %d Airflow DAGs in %s", len(dag_files), dag_output_dir)

    return {
        "assets_registered": len(specs),
        "git_sha": git_sha,
        "necessity": {
            "active": active,
            "transient": transient,
            "skipped": skipped,
        },
        "dag_files": dag_files,
    }


def plan_from_manifest(
    manifest_path: str,
    horizon_url: str,
    horizon_api_key: Optional[str] = None,
) -> dict:
    """
    Compute plan from manifest WITHOUT applying.
    Called by Trufflebox when admin clicks "Show Plan".

    Args:
        manifest_path: Path to manifest.json
        horizon_url: Internal horizon URL

    Returns:
        Plan result from horizon (changes, necessity, recomputation impact)
    """
    manifest = read_manifest(manifest_path)
    specs = [AssetSpec.from_dict(a) for a in manifest.get("assets", [])]

    client = HorizonClient(
        HorizonClientConfig(
            base_url=horizon_url,
            api_key=horizon_api_key,
        )
    )
    return client.propose_plan(specs, dry_run=True)
