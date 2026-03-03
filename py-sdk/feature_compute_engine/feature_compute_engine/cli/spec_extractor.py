"""
Extracts AssetSpecs from Python source files by importing them.

HOW IT WORKS:
- The @asset decorator registers metadata at import time
- We import each .py file in the notebooks directory
- The decorator fires, populating AssetRegistry
- We read the registry -- pure metadata, no Spark, no DataFrames
- The actual compute functions are NEVER called

REQUIREMENTS:
- Python 3.9+
- feature_compute_engine package installed
- pyspark installed with --no-deps (for import resolution only)
  OR notebook files guard Spark imports inside function bodies

NO REQUIREMENTS:
- No SparkSession
- No Databricks connectivity
- No JVM
- No network access
"""
from __future__ import annotations

import importlib.util
import logging
import sys
from pathlib import Path
from typing import List, Optional

from feature_compute_engine.decorators.registry import AssetRegistry
from feature_compute_engine.types.asset_spec import AssetSpec

logger = logging.getLogger(__name__)


class ExtractionError(Exception):
    """Raised when a notebook module fails to import."""

    def __init__(self, file_path: str, original_error: Exception):
        self.file_path = file_path
        self.original_error = original_error
        super().__init__(
            f"Failed to import '{file_path}': {original_error}\n"
            f"Hint: if this is a PySpark import error, ensure pyspark is installed "
            f"(pip install pyspark --no-deps) or move Spark imports inside function bodies."
        )


def extract_specs(
    notebook_dir: str,
    git_sha: Optional[str] = None,
    fail_on_error: bool = True,
) -> List[AssetSpec]:
    """
    Import all .py files in notebook_dir and extract AssetSpecs.

    Args:
        notebook_dir: Path to directory containing notebook .py files
        git_sha: Git SHA to stamp as asset version (optional)
        fail_on_error: If True, raise on first import failure.
                       If False, skip failed files and log warnings.

    Returns:
        List of AssetSpecs extracted from all successfully imported modules.
    """
    AssetRegistry.clear()

    notebook_path = Path(notebook_dir).resolve()
    if not notebook_path.exists():
        raise FileNotFoundError(f"Notebook directory not found: {notebook_dir}")

    py_files = sorted(notebook_path.glob("**/*.py"))
    py_files = [f for f in py_files if not f.name.startswith("_")]

    if not py_files:
        logger.warning("No .py files found in %s", notebook_dir)
        return []

    logger.info("Scanning %d files in %s", len(py_files), notebook_dir)

    errors: List[tuple] = []
    for py_file in py_files:
        relative = py_file.relative_to(notebook_path)
        module_name = f"_bharatml_extract_.{relative.stem}"

        logger.info("  Importing %s", relative)

        try:
            spec = importlib.util.spec_from_file_location(module_name, str(py_file))
            if spec is None or spec.loader is None:
                raise ImportError(f"Cannot create module spec for {py_file}")

            module = importlib.util.module_from_spec(spec)
            sys.modules[module_name] = module
            spec.loader.exec_module(module)

            count = sum(
                1
                for s in AssetRegistry.get_all().values()
                if s.notebook == module_name or s.notebook == relative.stem
            )
            logger.info("    -> %d assets registered", count)

        except Exception as e:
            if fail_on_error:
                raise ExtractionError(str(py_file), e)
            logger.warning("    -> FAILED: %s", e)
            errors.append((str(py_file), str(e)))

        finally:
            sys.modules.pop(module_name, None)

    all_specs = list(AssetRegistry.get_all().values())

    if git_sha:
        for asset_spec in all_specs:
            asset_spec.version = git_sha

    for asset_spec in all_specs:
        if asset_spec.notebook.startswith("_bharatml_extract_."):
            asset_spec.notebook = asset_spec.notebook.replace(
                "_bharatml_extract_.", ""
            )

    logger.info(
        "Extraction complete: %d assets, %d errors", len(all_specs), len(errors)
    )

    if errors:
        logger.warning("Failed files:")
        for filepath, error in errors:
            logger.warning("  %s: %s", filepath, error)

    return all_specs
