"""Necessity states and compute key types."""
from __future__ import annotations

import hashlib
from dataclasses import dataclass
from enum import Enum
from typing import Dict


class Necessity(str, Enum):
    """Determines whether and how an asset's output is used."""

    ACTIVE = "active"
    TRANSIENT = "transient"
    SKIPPED = "skipped"


@dataclass(frozen=True)
class ComputeKey:
    """
    Content-addressed identifier for a specific computation.
    If two ComputeKeys are equal, the output is guaranteed identical.
    """

    asset_version: str
    input_versions: Dict[str, str]
    params: Dict[str, str]
    env_fingerprint: str

    @property
    def key(self) -> str:
        parts = [
            self.asset_version,
            str(sorted(self.input_versions.items())),
            str(sorted(self.params.items())),
            self.env_fingerprint,
        ]
        raw = "|".join(parts)
        return hashlib.sha256(raw.encode()).hexdigest()[:16]

    def __str__(self) -> str:
        return self.key
