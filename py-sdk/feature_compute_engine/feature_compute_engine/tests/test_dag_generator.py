"""Tests for DAG Generator — generates Airflow DAG files from horizon registry."""
from __future__ import annotations

import os
import tempfile
from unittest.mock import MagicMock, patch

import pytest

from feature_compute_engine.airflow.dag_generator import (
    DAG_TEMPLATE,
    SCHEDULE_MAP,
    NotebookTriggerGroup,
    discover_trigger_groups,
    generate_all,
    generate_dag_file,
)


def _make_group(**overrides) -> NotebookTriggerGroup:
    defaults = dict(
        notebook="user_features",
        trigger_type="schedule_3h",
        asset_names=["fg.user_spend", "fg.user_clicks"],
        schedule="0 */3 * * *",
        notebook_path="/Repos/prod/features/user_features",
    )
    defaults.update(overrides)
    return NotebookTriggerGroup(**defaults)


class TestGenerateDagFile:
    def test_correct_dag_id(self) -> None:
        group = _make_group()
        content = generate_dag_file(group)

        assert 'dag_id="fce__user_features__schedule_3h"' in content

    def test_schedule_interval_for_cron(self) -> None:
        group = _make_group(schedule="0 */3 * * *")
        content = generate_dag_file(group)

        assert 'schedule_interval="0 */3 * * *"' in content

    def test_schedule_interval_none_for_upstream(self) -> None:
        group = _make_group(trigger_type="upstream", schedule=None)
        content = generate_dag_file(group)

        assert "schedule_interval=None" in content

    def test_max_active_runs_upstream(self) -> None:
        group = _make_group(trigger_type="upstream", schedule=None)
        content = generate_dag_file(group)

        assert "max_active_runs=1" in content

    def test_max_active_runs_schedule(self) -> None:
        group = _make_group(trigger_type="schedule_3h")
        content = generate_dag_file(group)

        assert "max_active_runs=3" in content

    def test_contains_asset_names(self) -> None:
        group = _make_group(
            asset_names=["fg.user_spend", "fg.user_clicks"]
        )
        content = generate_dag_file(group)

        assert "fg.user_spend" in content
        assert "fg.user_clicks" in content

    def test_notebook_path(self) -> None:
        group = _make_group(
            notebook_path="/Repos/prod/features/user_features"
        )
        content = generate_dag_file(group)

        assert 'notebook_path="/Repos/prod/features/user_features"' in content

    def test_valid_python(self) -> None:
        group = _make_group()
        content = generate_dag_file(group)

        compile(content, "<generated>", "exec")


class TestScheduleMap:
    def test_known_schedules(self) -> None:
        assert SCHEDULE_MAP["schedule_1h"] == "0 * * * *"
        assert SCHEDULE_MAP["schedule_3h"] == "0 */3 * * *"
        assert SCHEDULE_MAP["schedule_6h"] == "0 */6 * * *"
        assert SCHEDULE_MAP["schedule_12h"] == "0 */12 * * *"
        assert SCHEDULE_MAP["schedule_24h"] == "0 2 * * *"
        assert SCHEDULE_MAP["upstream"] is None


class TestDiscoverTriggerGroups:
    @patch(
        "feature_compute_engine.client.horizon_client.HorizonClient"
    )
    @patch(
        "feature_compute_engine.client.horizon_client.HorizonClientConfig"
    )
    def test_groups_by_notebook_and_trigger(
        self, MockConfig, MockClient
    ) -> None:
        mock_client = MagicMock()
        MockClient.return_value = mock_client
        mock_client.list_notebooks.return_value = [
            {
                "notebook": "user_features",
                "assets": [
                    {"name": "fg.user_spend", "trigger_type": "schedule_3h"},
                    {"name": "fg.user_clicks", "trigger_type": "schedule_3h"},
                    {"name": "fg.user_agg", "trigger_type": "upstream"},
                ],
            },
            {
                "notebook": "product_features",
                "assets": [
                    {"name": "fg.product_views", "trigger_type": "schedule_6h"},
                ],
            },
        ]

        groups = discover_trigger_groups("http://horizon:8080")

        assert len(groups) == 3
        by_key = {
            (g.notebook, g.trigger_type): g for g in groups
        }
        assert ("user_features", "schedule_3h") in by_key
        assert ("user_features", "upstream") in by_key
        assert ("product_features", "schedule_6h") in by_key

        uf_3h = by_key[("user_features", "schedule_3h")]
        assert set(uf_3h.asset_names) == {"fg.user_spend", "fg.user_clicks"}
        assert uf_3h.schedule == "0 */3 * * *"


class TestGenerateAll:
    @patch(
        "feature_compute_engine.airflow.dag_generator.discover_trigger_groups"
    )
    def test_creates_correct_number_of_files(
        self, mock_discover
    ) -> None:
        mock_discover.return_value = [
            _make_group(
                notebook="user_features",
                trigger_type="schedule_3h",
            ),
            _make_group(
                notebook="user_features",
                trigger_type="upstream",
                schedule=None,
            ),
            _make_group(
                notebook="product_features",
                trigger_type="schedule_6h",
                schedule="0 */6 * * *",
            ),
        ]

        with tempfile.TemporaryDirectory() as tmpdir:
            written = generate_all(
                "http://horizon:8080", tmpdir
            )

            assert len(written) == 3
            filenames = [os.path.basename(f) for f in written]
            assert "fce__user_features__schedule_3h.py" in filenames
            assert "fce__user_features__upstream.py" in filenames
            assert "fce__product_features__schedule_6h.py" in filenames

            for filepath in written:
                assert os.path.exists(filepath)
                with open(filepath) as f:
                    content = f.read()
                assert len(content) > 0
                compile(content, filepath, "exec")

    @patch(
        "feature_compute_engine.airflow.dag_generator.discover_trigger_groups"
    )
    def test_creates_output_directory(self, mock_discover) -> None:
        mock_discover.return_value = []

        with tempfile.TemporaryDirectory() as tmpdir:
            out = os.path.join(tmpdir, "subdir", "dags")
            generate_all("http://horizon:8080", out)
            assert os.path.isdir(out)

    @patch(
        "feature_compute_engine.airflow.dag_generator.discover_trigger_groups"
    )
    def test_empty_registry(self, mock_discover) -> None:
        mock_discover.return_value = []

        with tempfile.TemporaryDirectory() as tmpdir:
            written = generate_all(
                "http://horizon:8080", tmpdir
            )
            assert written == []
