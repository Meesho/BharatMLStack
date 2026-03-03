"""Tests for AirflowTrigger — triggers Airflow DAGs via REST API."""
from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from feature_compute_engine.client.airflow_trigger import (
    AirflowTrigger,
    AirflowTriggerConfig,
)


def _make_trigger(**overrides) -> AirflowTrigger:
    defaults = dict(
        base_url="http://airflow:8080",
        timeout_seconds=5,
    )
    defaults.update(overrides)
    return AirflowTrigger(AirflowTriggerConfig(**defaults))


def _mock_response(status_code: int = 200, json_data=None):
    resp = MagicMock()
    resp.status_code = status_code
    resp.json.return_value = json_data or {}
    resp.raise_for_status.side_effect = (
        None if status_code < 400 else Exception(f"HTTP {status_code}")
    )
    return resp


class TestAirflowTriggerConfig:
    def test_defaults(self) -> None:
        cfg = AirflowTriggerConfig(base_url="http://airflow:8080")
        assert cfg.username is None
        assert cfg.password is None
        assert cfg.timeout_seconds == 15


class TestAirflowTriggerSession:
    @patch("requests.Session")
    def test_session_with_auth(self, MockSession) -> None:
        mock_session = MagicMock()
        mock_session.auth = None
        MockSession.return_value = mock_session

        trigger = _make_trigger(username="admin", password="admin")
        session = trigger.session

        assert session is mock_session
        assert mock_session.auth == ("admin", "admin")

    @patch("requests.Session")
    def test_session_without_auth(self, MockSession) -> None:
        mock_session = MagicMock()
        mock_session.auth = None
        MockSession.return_value = mock_session

        trigger = _make_trigger()
        _ = trigger.session

        assert mock_session.auth is None


class TestTriggerDag:
    def test_posts_to_correct_url(self) -> None:
        trigger = _make_trigger()
        trigger._session = MagicMock()
        trigger._session.post.return_value = _mock_response(
            200, {"dag_run_id": "run-123"}
        )

        run_id = trigger.trigger_dag("fce__user_features__schedule_3h")

        assert run_id == "run-123"
        call_args = trigger._session.post.call_args
        assert (
            "http://airflow:8080/api/v1/dags/fce__user_features__schedule_3h/dagRuns"
            == call_args[0][0]
        )

    def test_passes_conf_and_logical_date(self) -> None:
        trigger = _make_trigger()
        trigger._session = MagicMock()
        trigger._session.post.return_value = _mock_response(
            200, {"dag_run_id": "run-456"}
        )

        run_id = trigger.trigger_dag(
            "my_dag",
            conf={"partition": "2025-01-15"},
            logical_date="2025-01-15T00:00:00Z",
        )

        assert run_id == "run-456"
        call_args = trigger._session.post.call_args
        payload = call_args[1]["json"]
        assert payload["conf"] == {"partition": "2025-01-15"}
        assert payload["logical_date"] == "2025-01-15T00:00:00Z"

    def test_empty_payload_when_no_conf(self) -> None:
        trigger = _make_trigger()
        trigger._session = MagicMock()
        trigger._session.post.return_value = _mock_response(
            200, {"dag_run_id": "run-789"}
        )

        trigger.trigger_dag("my_dag")

        call_args = trigger._session.post.call_args
        payload = call_args[1]["json"]
        assert payload == {}

    def test_returns_unknown_when_no_run_id(self) -> None:
        trigger = _make_trigger()
        trigger._session = MagicMock()
        trigger._session.post.return_value = _mock_response(200, {})

        run_id = trigger.trigger_dag("my_dag")
        assert run_id == "unknown"


class TestTriggerFromActions:
    def test_maps_actions_to_dag_ids(self) -> None:
        trigger = _make_trigger()
        trigger._session = MagicMock()
        trigger._session.post.return_value = _mock_response(
            200, {"dag_run_id": "run-1"}
        )

        actions = [
            {
                "notebook": "user_features",
                "trigger_type": "upstream",
                "partition": "2025-01-15",
                "asset_name": "fg.user_spend",
            },
            {
                "notebook": "product_features",
                "trigger_type": "upstream",
                "partition": "2025-01-15",
                "asset_name": "fg.product_clicks",
            },
        ]

        run_ids = trigger.trigger_from_actions(actions)

        assert len(run_ids) == 2
        urls = [
            call[0][0] for call in trigger._session.post.call_args_list
        ]
        assert any("fce__user_features__upstream" in u for u in urls)
        assert any("fce__product_features__upstream" in u for u in urls)

    def test_failure_on_one_dag_continues(self) -> None:
        trigger = _make_trigger()
        trigger._session = MagicMock()

        success_resp = _mock_response(200, {"dag_run_id": "run-ok"})
        fail_resp = _mock_response(500)
        trigger._session.post.side_effect = [fail_resp, success_resp]

        actions = [
            {
                "notebook": "broken",
                "trigger_type": "upstream",
                "partition": "2025-01-15",
            },
            {
                "notebook": "working",
                "trigger_type": "upstream",
                "partition": "2025-01-15",
            },
        ]

        run_ids = trigger.trigger_from_actions(actions)

        assert len(run_ids) == 1
        assert run_ids[0] == "run-ok"

    def test_empty_actions(self) -> None:
        trigger = _make_trigger()
        trigger._session = MagicMock()

        run_ids = trigger.trigger_from_actions([])
        assert run_ids == []
