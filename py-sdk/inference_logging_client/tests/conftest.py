"""Shared pytest fixtures for inference_logging_client tests."""

import pytest


@pytest.fixture(scope="session")
def spark():
    """Shared SparkSession for tests that need local Spark (E2E, from_json fallback)."""
    from pyspark.sql import SparkSession
    return SparkSession.builder.master("local[2]").appName("test_decode_pipeline").getOrCreate()
