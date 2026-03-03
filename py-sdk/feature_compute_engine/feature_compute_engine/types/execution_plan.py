"""
Execution plan types — received from horizon, consumed by NotebookRuntime.

These types represent the control plane's decisions about what to execute
in a given notebook invocation.
"""
from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Dict, List, Optional


class AssetAction:
    """Constants for asset execution actions."""

    EXECUTE = "execute"
    SKIP_CACHED = "skip_cached"
    SKIP_UNNECESSARY = "skip_unnecessary"


@dataclass
class AssetExecutionPlan:
    """Plan for a single asset within a notebook invocation."""

    asset_name: str
    action: str
    necessity: str
    compute_key: Optional[str] = None
    input_bindings: Dict[str, str] = field(default_factory=dict)
    artifact_path: Optional[str] = None
    existing_artifact: Optional[str] = None
    checks: List[str] = field(default_factory=list)
    reason: Optional[str] = None

    @property
    def should_execute(self) -> bool:
        return self.action == AssetAction.EXECUTE

    @property
    def should_publish(self) -> bool:
        return self.necessity == "active" and self.action == AssetAction.EXECUTE


@dataclass
class NotebookExecutionPlan:
    """
    Complete execution plan for a single notebook invocation.
    Passed as a JSON parameter to the Databricks notebook.
    """

    notebook: str
    trigger_type: str
    partition: str
    assets: List[AssetExecutionPlan] = field(default_factory=list)
    shared_inputs: Dict[str, str] = field(default_factory=dict)

    @property
    def assets_to_execute(self) -> List[AssetExecutionPlan]:
        return [a for a in self.assets if a.should_execute]

    @property
    def has_work(self) -> bool:
        return len(self.assets_to_execute) > 0

    def to_json(self) -> str:
        return json.dumps(self._to_dict())

    @classmethod
    def from_json(cls, raw: str) -> NotebookExecutionPlan:
        d = json.loads(raw)
        return cls(
            notebook=d["notebook"],
            trigger_type=d["trigger_type"],
            partition=d["partition"],
            assets=[
                AssetExecutionPlan(
                    asset_name=a["asset_name"],
                    action=a["action"],
                    necessity=a.get("necessity", "active"),
                    compute_key=a.get("compute_key"),
                    input_bindings=a.get("input_bindings", {}),
                    artifact_path=a.get("artifact_path"),
                    existing_artifact=a.get("existing_artifact"),
                    checks=a.get("checks", []),
                    reason=a.get("reason"),
                )
                for a in d.get("assets", [])
            ],
            shared_inputs=d.get("shared_inputs", {}),
        )

    def _to_dict(self) -> dict:
        return {
            "notebook": self.notebook,
            "trigger_type": self.trigger_type,
            "partition": self.partition,
            "assets": [
                {
                    "asset_name": a.asset_name,
                    "action": a.action,
                    "necessity": a.necessity,
                    "compute_key": a.compute_key,
                    "input_bindings": a.input_bindings,
                    "artifact_path": a.artifact_path,
                    "existing_artifact": a.existing_artifact,
                    "checks": a.checks,
                    "reason": a.reason,
                }
                for a in self.assets
            ],
            "shared_inputs": self.shared_inputs,
        }
