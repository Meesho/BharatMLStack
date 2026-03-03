"""Tests for the CLI apply module."""
from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from feature_compute_engine.cli.apply import apply_manifest, plan_from_manifest


@pytest.fixture
def manifest_file(tmp_path):
    """Create a sample manifest file and return its path."""
    manifest = {
        "version": "1",
        "git_sha": "abc123",
        "assets": [
            {
                "name": "fg.user_spend",
                "entity": "user",
                "entity_key": "user_id",
                "notebook": "user_features",
                "partition": "ds",
                "trigger": "schedule_3h",
                "schedule": "3h",
                "serving": True,
                "incremental": False,
                "freshness": None,
                "inputs": [{"name": "silver.orders", "partition": "ds", "window": None, "type": "internal"}],
                "checks": [],
                "version": "abc123",
            },
            {
                "name": "fg.user_orders",
                "entity": "user",
                "entity_key": "user_id",
                "notebook": "user_features",
                "partition": "ds",
                "trigger": "upstream",
                "schedule": None,
                "serving": False,
                "incremental": False,
                "freshness": None,
                "inputs": [],
                "checks": [],
                "version": "abc123",
            },
        ],
        "summary": {"total_assets": 2},
    }
    path = tmp_path / "manifest.json"
    with open(path, "w") as f:
        json.dump(manifest, f)
    return str(path)


class TestApplyManifest:
    @patch("feature_compute_engine.cli.apply.generate_all")
    @patch("feature_compute_engine.cli.apply.HorizonClient")
    def test_registers_assets_and_resolves_necessity(
        self, mock_client_cls, mock_gen_all, manifest_file
    ):
        mock_client = MagicMock()
        mock_client_cls.return_value = mock_client
        mock_client.register_assets.return_value = {"status": "ok"}
        mock_client.get_necessity.return_value = {
            "fg.user_spend": "active",
            "fg.user_orders": "transient",
        }

        result = apply_manifest(
            manifest_path=manifest_file,
            horizon_url="http://horizon.internal:8080",
        )

        mock_client.register_assets.assert_called_once()
        registered_specs = mock_client.register_assets.call_args[0][0]
        assert len(registered_specs) == 2

        mock_client.get_necessity.assert_called_once()

        assert result["assets_registered"] == 2
        assert result["git_sha"] == "abc123"
        assert result["necessity"]["active"] == 1
        assert result["necessity"]["transient"] == 1
        assert result["necessity"]["skipped"] == 0
        assert result["dag_files"] == []

    @patch("feature_compute_engine.cli.apply.generate_all")
    @patch("feature_compute_engine.cli.apply.HorizonClient")
    def test_generates_dags_when_output_dir_provided(
        self, mock_client_cls, mock_gen_all, manifest_file, tmp_path
    ):
        mock_client = MagicMock()
        mock_client_cls.return_value = mock_client
        mock_client.register_assets.return_value = {"status": "ok"}
        mock_client.get_necessity.return_value = {}

        dag_dir = str(tmp_path / "dags")
        mock_gen_all.return_value = [f"{dag_dir}/dag1.py", f"{dag_dir}/dag2.py"]

        result = apply_manifest(
            manifest_path=manifest_file,
            horizon_url="http://horizon.internal:8080",
            dag_output_dir=dag_dir,
        )

        mock_gen_all.assert_called_once_with(
            "http://horizon.internal:8080", dag_dir, None
        )
        assert len(result["dag_files"]) == 2

    @patch("feature_compute_engine.cli.apply.generate_all")
    @patch("feature_compute_engine.cli.apply.HorizonClient")
    def test_skips_dags_when_no_output_dir(
        self, mock_client_cls, mock_gen_all, manifest_file
    ):
        mock_client = MagicMock()
        mock_client_cls.return_value = mock_client
        mock_client.register_assets.return_value = {"status": "ok"}
        mock_client.get_necessity.return_value = {}

        apply_manifest(
            manifest_path=manifest_file,
            horizon_url="http://horizon.internal:8080",
        )

        mock_gen_all.assert_not_called()


class TestPlanFromManifest:
    @patch("feature_compute_engine.cli.apply.HorizonClient")
    def test_calls_propose_plan_with_dry_run(self, mock_client_cls, manifest_file):
        mock_client = MagicMock()
        mock_client_cls.return_value = mock_client
        mock_client.propose_plan.return_value = {
            "changes": [
                {"asset_name": "fg.user_spend", "change_type": "added"},
            ],
            "total_partitions": 30,
            "total_estimated_cost_usd": 1.50,
        }

        result = plan_from_manifest(
            manifest_path=manifest_file,
            horizon_url="http://horizon.internal:8080",
        )

        mock_client.propose_plan.assert_called_once()
        call_args = mock_client.propose_plan.call_args
        assert call_args[1]["dry_run"] is True
        specs = call_args[0][0]
        assert len(specs) == 2

        assert len(result["changes"]) == 1
        assert result["total_partitions"] == 30

    @patch("feature_compute_engine.cli.apply.HorizonClient")
    def test_passes_api_key(self, mock_client_cls, manifest_file):
        mock_client = MagicMock()
        mock_client_cls.return_value = mock_client
        mock_client.propose_plan.return_value = {"changes": []}

        plan_from_manifest(
            manifest_path=manifest_file,
            horizon_url="http://horizon.internal:8080",
            horizon_api_key="secret-key",
        )

        config = mock_client_cls.call_args[0][0]
        assert config.api_key == "secret-key"
