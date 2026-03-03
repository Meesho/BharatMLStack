"""
Global registry of all @asset-decorated functions.

The registry is populated at import time when Python loads modules
containing @asset decorators. NotebookRuntime reads this registry
to discover which assets are available in the current notebook.
"""
from __future__ import annotations

from typing import Dict, List

from ..types.asset_spec import AssetSpec


class AssetRegistry:
    """
    Singleton registry of all registered assets.

    Thread-safe for read; write happens at import time (single-threaded).
    """

    _assets: Dict[str, AssetSpec] = {}

    @classmethod
    def register(cls, spec: AssetSpec) -> None:
        if spec.name in cls._assets:
            existing = cls._assets[spec.name]
            if existing.notebook != spec.notebook:
                raise ValueError(
                    f"Asset '{spec.name}' is already registered in notebook "
                    f"'{existing.notebook}', cannot register again in "
                    f"'{spec.notebook}'"
                )
        cls._assets[spec.name] = spec

    @classmethod
    def get(cls, name: str) -> AssetSpec:
        if name not in cls._assets:
            raise KeyError(f"Asset '{name}' is not registered")
        return cls._assets[name]

    @classmethod
    def get_all(cls) -> Dict[str, AssetSpec]:
        return dict(cls._assets)

    @classmethod
    def get_by_notebook(cls, notebook: str) -> Dict[str, AssetSpec]:
        return {
            name: spec
            for name, spec in cls._assets.items()
            if spec.notebook == notebook
        }

    @classmethod
    def get_by_entity(cls, entity: str) -> Dict[str, AssetSpec]:
        return {
            name: spec
            for name, spec in cls._assets.items()
            if spec.entity == entity
        }

    @classmethod
    def get_names(cls) -> List[str]:
        return list(cls._assets.keys())

    @classmethod
    def clear(cls) -> None:
        """Clear registry. Used in tests."""
        cls._assets.clear()

    @classmethod
    def count(cls) -> int:
        return len(cls._assets)
