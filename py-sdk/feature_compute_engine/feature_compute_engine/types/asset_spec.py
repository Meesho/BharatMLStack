"""
Core types for the Feature Computation Platform.

AssetSpec is the canonical definition of a feature computation asset.
It captures what to compute, what it depends on, and how it's triggered.
"""
from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Callable, Dict, List, Optional


@dataclass(frozen=True)
class Input:
    """
    Declares a dependency on another asset or external data source.

    Usage:
        Input("silver.orders", partition="ds")
        Input("silver.orders", partition="ds", window=7)
        Input("fg.user_spend", partition="ds")
    """

    name: str
    partition: str = "ds"
    window: Optional[int] = None
    input_type: str = "internal"  # "internal" (asset->asset) | "external" (raw source)

    def __post_init__(self) -> None:
        if not self.name:
            raise ValueError("Input name cannot be empty")
        if self.window is not None and self.window < 1:
            raise ValueError(f"Window size must be >= 1, got {self.window}")


@dataclass
class AssetSpec:
    """
    Complete specification of a feature computation asset.

    This is what gets registered with horizon's Asset Registry.
    It contains everything the control plane needs to build the DAG,
    resolve necessity, compute cache keys, and generate execution plans.
    """

    name: str
    entity: str
    entity_key: str
    notebook: str
    partition: str = "ds"
    trigger: str = "upstream"
    schedule: Optional[str] = None
    serving: bool = True
    incremental: bool = False
    freshness: Optional[str] = None
    inputs: List[Input] = field(default_factory=list)
    checks: List[str] = field(default_factory=list)
    version: Optional[str] = None

    _compute_fn: Optional[Callable] = field(default=None, repr=False, compare=False)

    def __post_init__(self) -> None:
        if not self.name:
            raise ValueError("Asset name cannot be empty")
        if not self.entity:
            raise ValueError("Entity cannot be empty")
        if not self.entity_key:
            raise ValueError("Entity key cannot be empty")
        if self.trigger.startswith("schedule") and not self.schedule:
            raise ValueError(
                f"Schedule is required for trigger type '{self.trigger}'"
            )

        if self.schedule and not self.trigger.startswith("schedule"):
            self.trigger = f"schedule_{self.schedule}"

    @property
    def input_names(self) -> List[str]:
        return [inp.name for inp in self.inputs]

    @property
    def internal_inputs(self) -> List[Input]:
        return [inp for inp in self.inputs if inp.input_type == "internal"]

    @property
    def external_inputs(self) -> List[Input]:
        return [inp for inp in self.inputs if inp.input_type == "external"]

    def to_dict(self) -> Dict:
        """Serialize to dict for JSON transport (excludes compute function)."""
        return {
            "name": self.name,
            "entity": self.entity,
            "entity_key": self.entity_key,
            "notebook": self.notebook,
            "partition": self.partition,
            "trigger": self.trigger,
            "schedule": self.schedule,
            "serving": self.serving,
            "incremental": self.incremental,
            "freshness": self.freshness,
            "inputs": [
                {
                    "name": inp.name,
                    "partition": inp.partition,
                    "window": inp.window,
                    "type": inp.input_type,
                }
                for inp in self.inputs
            ],
            "checks": self.checks,
            "version": self.version,
        }

    def to_json(self) -> str:
        return json.dumps(self.to_dict())

    @classmethod
    def from_dict(cls, d: Dict) -> AssetSpec:
        inputs = [
            Input(
                name=inp["name"],
                partition=inp.get("partition", "ds"),
                window=inp.get("window"),
                input_type=inp.get("type", "internal"),
            )
            for inp in d.get("inputs", [])
        ]
        return cls(
            name=d["name"],
            entity=d["entity"],
            entity_key=d["entity_key"],
            notebook=d["notebook"],
            partition=d.get("partition", "ds"),
            trigger=d.get("trigger", "upstream"),
            schedule=d.get("schedule"),
            serving=d.get("serving", True),
            incremental=d.get("incremental", False),
            freshness=d.get("freshness"),
            inputs=inputs,
            checks=d.get("checks", []),
            version=d.get("version"),
        )
