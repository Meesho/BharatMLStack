"""Result types for asset execution within a notebook run."""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


@dataclass
class AssetResult:
    """Result of executing (or skipping) a single asset."""

    asset_name: str
    action: str  # "executed" | "skipped_cached" | "skipped_unnecessary" | "failed"
    compute_key: Optional[str] = None
    artifact_path: Optional[str] = None
    error: Optional[str] = None
    reason: Optional[str] = None
    row_count: Optional[int] = None
    duration_seconds: Optional[float] = None
    dq_results: Dict[str, bool] = field(default_factory=dict)

    @property
    def succeeded(self) -> bool:
        return self.action == "executed"

    @property
    def skipped(self) -> bool:
        return self.action in ("skipped_cached", "skipped_unnecessary")

    @property
    def failed(self) -> bool:
        return self.action == "failed"

    def to_dict(self) -> dict:
        return {
            "asset_name": self.asset_name,
            "action": self.action,
            "compute_key": self.compute_key,
            "artifact_path": self.artifact_path,
            "error": self.error,
            "reason": self.reason,
            "row_count": self.row_count,
            "duration_seconds": self.duration_seconds,
            "dq_results": self.dq_results,
        }


@dataclass
class NotebookRunResult:
    """Aggregate result of a notebook invocation."""

    notebook: str
    trigger_type: str
    partition: str
    results: List[AssetResult] = field(default_factory=list)

    @property
    def all_succeeded(self) -> bool:
        return all(r.succeeded or r.skipped for r in self.results)

    @property
    def executed_count(self) -> int:
        return sum(1 for r in self.results if r.succeeded)

    @property
    def failed_count(self) -> int:
        return sum(1 for r in self.results if r.failed)

    @property
    def skipped_count(self) -> int:
        return sum(1 for r in self.results if r.skipped)

    def to_dict(self) -> dict:
        return {
            "notebook": self.notebook,
            "trigger_type": self.trigger_type,
            "partition": self.partition,
            "results": [r.to_dict() for r in self.results],
            "summary": {
                "executed": self.executed_count,
                "failed": self.failed_count,
                "skipped": self.skipped_count,
            },
        }
