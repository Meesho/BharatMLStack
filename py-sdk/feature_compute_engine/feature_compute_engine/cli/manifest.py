"""
Manifest generation -- produces the .bharatml/manifest.json file.

The manifest is a pure JSON file containing all extracted AssetSpecs
plus summary metadata. It is committed to the git repo and serves as
the bridge between CI (external) and horizon (internal).

The manifest is:
- Deterministic: same code -> same manifest (sorted, stable)
- Self-contained: everything horizon needs to compute a plan
- Auditable: committed to git, visible in PR diff
- Safe: no secrets, no internal URLs, just data definitions
"""
from __future__ import annotations

import json
import logging
from datetime import datetime, timezone
from pathlib import Path
from typing import List, Optional

from feature_compute_engine.types.asset_spec import AssetSpec

from .spec_extractor import extract_specs

logger = logging.getLogger(__name__)

MANIFEST_DIR = ".bharatml"
MANIFEST_FILE = "manifest.json"
MANIFEST_VERSION = "1"


def generate_manifest(
    notebook_dir: str,
    output_dir: Optional[str] = None,
    git_sha: Optional[str] = None,
    git_ref: Optional[str] = None,
) -> dict:
    """
    Extract AssetSpecs from notebooks and write manifest.json.

    Args:
        notebook_dir: Path to notebook .py files
        output_dir: Where to write .bharatml/manifest.json (default: cwd)
        git_sha: Current git commit SHA
        git_ref: Current git branch/ref

    Returns:
        The manifest dict (also written to disk)
    """
    specs = extract_specs(notebook_dir, git_sha=git_sha)
    manifest = build_manifest(specs, git_sha=git_sha, git_ref=git_ref)

    if output_dir is None:
        output_dir = "."
    manifest_path = Path(output_dir) / MANIFEST_DIR / MANIFEST_FILE
    manifest_path.parent.mkdir(parents=True, exist_ok=True)

    with open(manifest_path, "w") as f:
        json.dump(manifest, f, indent=2, sort_keys=False)

    logger.info("Manifest written to %s (%d assets)", manifest_path, len(specs))
    return manifest


def build_manifest(
    specs: List[AssetSpec],
    git_sha: Optional[str] = None,
    git_ref: Optional[str] = None,
) -> dict:
    """Build the manifest dict from extracted specs."""
    sorted_specs = sorted(specs, key=lambda s: s.name)

    by_entity: dict = {}
    by_trigger: dict = {}
    serving_count = 0
    non_serving_count = 0

    for spec in sorted_specs:
        by_entity[spec.entity] = by_entity.get(spec.entity, 0) + 1
        by_trigger[spec.trigger] = by_trigger.get(spec.trigger, 0) + 1
        if spec.serving:
            serving_count += 1
        else:
            non_serving_count += 1

    return {
        "version": MANIFEST_VERSION,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "generated_by": "bharatml-cli",
        "git_sha": git_sha,
        "git_ref": git_ref,
        "assets": [spec.to_dict() for spec in sorted_specs],
        "summary": {
            "total_assets": len(sorted_specs),
            "by_entity": dict(sorted(by_entity.items())),
            "by_trigger": dict(sorted(by_trigger.items())),
            "serving_count": serving_count,
            "non_serving_count": non_serving_count,
        },
    }


def read_manifest(manifest_path: str) -> dict:
    """Read an existing manifest file."""
    with open(manifest_path, "r") as f:
        return json.load(f)


def diff_manifests(current: dict, proposed: dict) -> dict:
    """
    Compare two manifests and produce a local diff.
    This is a lightweight client-side diff -- NOT the full horizon impact analysis.
    Used for the CI PR comment (quick summary without calling horizon).
    """
    current_assets = {a["name"]: a for a in current.get("assets", [])}
    proposed_assets = {a["name"]: a for a in proposed.get("assets", [])}

    added = [name for name in proposed_assets if name not in current_assets]
    removed = [name for name in current_assets if name not in proposed_assets]
    modified = []

    for name in proposed_assets:
        if name in current_assets:
            if proposed_assets[name] != current_assets[name]:
                curr = current_assets[name]
                prop = proposed_assets[name]
                changes = []
                curr_inputs = {i["name"] for i in curr.get("inputs", [])}
                prop_inputs = {i["name"] for i in prop.get("inputs", [])}
                if curr_inputs != prop_inputs:
                    changes.append("inputs_changed")
                if curr.get("schedule") != prop.get("schedule"):
                    changes.append("schedule_changed")
                if curr.get("serving") != prop.get("serving"):
                    changes.append("serving_changed")
                if curr.get("version") != prop.get("version"):
                    changes.append("code_changed")
                if changes:
                    modified.append({"name": name, "changes": changes})

    return {
        "added": added,
        "removed": removed,
        "modified": modified,
        "has_changes": bool(added or removed or modified),
    }
