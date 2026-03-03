"""Tests for HorizonClient — HTTP client for the horizon FCE API."""
from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from feature_compute_engine.client.horizon_client import (
    HorizonClient,
    HorizonClientConfig,
    HorizonClientError,
)
from feature_compute_engine.types.asset_spec import AssetSpec, Input
from feature_compute_engine.types.execution_plan import (
    AssetAction,
    NotebookExecutionPlan,
)


def _make_config(**overrides) -> HorizonClientConfig:
    defaults = dict(
        base_url="http://horizon:8080",
        api_prefix="/api/v1/fce",
        timeout_seconds=5,
        max_retries=3,
        retry_delay_seconds=0.01,
    )
    defaults.update(overrides)
    return HorizonClientConfig(**defaults)


def _make_client(**overrides) -> HorizonClient:
    return HorizonClient(_make_config(**overrides))


def _mock_response(status_code: int = 200, json_data=None, text: str = ""):
    resp = MagicMock()
    resp.status_code = status_code
    resp.content = json.dumps(json_data).encode() if json_data is not None else b""
    resp.json.return_value = json_data if json_data is not None else {}
    resp.text = text or json.dumps(json_data) if json_data else text
    return resp


class TestHorizonClientConfig:
    def test_defaults(self) -> None:
        cfg = HorizonClientConfig(base_url="http://localhost:8080")
        assert cfg.api_prefix == "/api/v1/fce"
        assert cfg.api_key is None
        assert cfg.timeout_seconds == 30
        assert cfg.max_retries == 3
        assert cfg.retry_delay_seconds == 1.0


class TestHorizonClientError:
    def test_attributes(self) -> None:
        err = HorizonClientError(404, "not found", "GET /foo")
        assert err.status_code == 404
        assert err.endpoint == "GET /foo"
        assert "404" in str(err)
        assert "GET /foo" in str(err)
        assert "not found" in str(err)


class TestHorizonClientSession:
    @patch("requests.Session")
    def test_session_created_with_headers(self, MockSession) -> None:
        mock_session = MagicMock()
        MockSession.return_value = mock_session

        client = _make_client(api_key="test-key")
        session = client.session

        assert session is mock_session
        mock_session.headers.update.assert_called_once()
        mock_session.headers.__setitem__.assert_called_with(
            "Authorization", "Bearer test-key"
        )

    @patch("requests.Session")
    def test_session_no_auth_header_without_key(self, MockSession) -> None:
        mock_session = MagicMock()
        MockSession.return_value = mock_session

        client = _make_client(api_key=None)
        _ = client.session

        mock_session.headers.__setitem__.assert_not_called()

    @patch("requests.Session")
    def test_session_cached(self, MockSession) -> None:
        mock_session = MagicMock()
        MockSession.return_value = mock_session

        client = _make_client()
        s1 = client.session
        s2 = client.session

        assert s1 is s2
        MockSession.assert_called_once()


class TestHorizonClientRequest:
    def _client_with_mock_session(self, response=None, side_effect=None):
        client = _make_client()
        mock_session = MagicMock()
        if side_effect:
            mock_session.request.side_effect = side_effect
        elif response:
            mock_session.request.return_value = response
        client._session = mock_session
        return client, mock_session

    def test_successful_request(self) -> None:
        resp = _mock_response(200, {"key": "value"})
        client, mock_session = self._client_with_mock_session(response=resp)

        result = client._request("GET", "/test")

        assert result == {"key": "value"}
        mock_session.request.assert_called_once()

    def test_empty_response(self) -> None:
        resp = _mock_response(200)
        resp.content = b""
        client, _ = self._client_with_mock_session(response=resp)

        result = client._request("GET", "/test")
        assert result == {}

    def test_4xx_raises_horizon_error(self) -> None:
        resp = _mock_response(404, text="not found")
        client, _ = self._client_with_mock_session(response=resp)

        with pytest.raises(HorizonClientError) as exc_info:
            client._request("GET", "/missing")

        assert exc_info.value.status_code == 404
        assert exc_info.value.endpoint == "GET /missing"

    def test_5xx_raises_horizon_error(self) -> None:
        resp = _mock_response(500, text="internal error")
        client, _ = self._client_with_mock_session(response=resp)

        with pytest.raises(HorizonClientError) as exc_info:
            client._request("POST", "/broken")

        assert exc_info.value.status_code == 500

    def test_horizon_error_not_retried(self) -> None:
        resp = _mock_response(400, text="bad request")
        client, mock_session = self._client_with_mock_session(response=resp)

        with pytest.raises(HorizonClientError):
            client._request("POST", "/bad")

        assert mock_session.request.call_count == 1

    def test_transient_error_retried(self) -> None:
        success_resp = _mock_response(200, {"ok": True})
        client, mock_session = self._client_with_mock_session(
            side_effect=[ConnectionError("conn refused"), success_resp]
        )

        result = client._request("GET", "/flaky")

        assert result == {"ok": True}
        assert mock_session.request.call_count == 2

    def test_all_retries_exhausted(self) -> None:
        client, mock_session = self._client_with_mock_session(
            side_effect=ConnectionError("conn refused")
        )

        with pytest.raises(RuntimeError, match="Failed after 3 retries"):
            client._request("GET", "/down")

        assert mock_session.request.call_count == 3

    def test_url_construction(self) -> None:
        client = _make_client(base_url="http://horizon:8080")
        url = client._url("/execution-plan")
        assert url == "http://horizon:8080/api/v1/fce/execution-plan"


class TestGetExecutionPlan:
    def test_happy_path(self) -> None:
        plan_data = {
            "notebook": "user_features",
            "trigger_type": "schedule_3h",
            "partition": "2025-01-15",
            "assets": [
                {
                    "asset_name": "fg.user_spend",
                    "action": "execute",
                    "necessity": "active",
                    "compute_key": "abc123",
                    "artifact_path": "/_artifacts/fg.user_spend/2025-01-15/abc123/",
                },
                {
                    "asset_name": "fg.user_clicks",
                    "action": "skip_cached",
                    "necessity": "active",
                    "compute_key": "def456",
                },
            ],
        }
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(200, plan_data)

        plan = client.get_execution_plan(
            notebook="user_features",
            trigger_type="schedule_3h",
            partition="2025-01-15",
        )

        assert isinstance(plan, NotebookExecutionPlan)
        assert plan.notebook == "user_features"
        assert plan.partition == "2025-01-15"
        assert len(plan.assets) == 2
        assert len(plan.assets_to_execute) == 1
        assert plan.has_work is True


class TestReportAssetReady:
    def test_returns_trigger_actions(self) -> None:
        response_data = {
            "trigger_actions": [
                {
                    "notebook": "product_features",
                    "trigger_type": "upstream",
                    "partition": "2025-01-15",
                    "asset_name": "fg.user_spend",
                },
            ]
        }
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(200, response_data)

        actions = client.report_asset_ready(
            asset_name="fg.user_spend",
            partition="2025-01-15",
            compute_key="abc123",
            artifact_path="/_artifacts/fg.user_spend/2025-01-15/abc123/",
        )

        assert len(actions) == 1
        assert actions[0]["notebook"] == "product_features"

    def test_empty_trigger_actions(self) -> None:
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(200, {})

        actions = client.report_asset_ready(
            asset_name="fg.leaf",
            partition="2025-01-15",
            compute_key="xyz",
            artifact_path="/path",
        )

        assert actions == []


class TestReportAssetFailed:
    def test_happy_path(self) -> None:
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(200, {})

        client.report_asset_failed(
            asset_name="fg.broken",
            partition="2025-01-15",
            error="OOM",
        )

        call_args = client._session.request.call_args
        assert call_args[0][0] == "POST"
        payload = call_args[1]["json"]
        assert payload["asset_name"] == "fg.broken"
        assert payload["error"] == "OOM"


class TestRegisterAssets:
    def test_serializes_specs(self) -> None:
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(
            200, {"registered": 1}
        )

        specs = [
            AssetSpec(
                name="fg.user_spend",
                entity="user",
                entity_key="user_id",
                notebook="user_features",
                schedule="3h",
                inputs=[Input("silver.orders")],
            )
        ]
        result = client.register_assets(specs)

        assert result == {"registered": 1}
        call_args = client._session.request.call_args
        payload = call_args[1]["json"]
        assert len(payload["assets"]) == 1
        assert payload["assets"][0]["name"] == "fg.user_spend"


class TestProposePlan:
    def test_dry_run(self) -> None:
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(
            200, {"plan_id": "plan-1", "changes": []}
        )

        specs = [
            AssetSpec(
                name="fg.test",
                entity="user",
                entity_key="user_id",
                notebook="nb",
                schedule="3h",
            )
        ]
        result = client.propose_plan(specs, dry_run=True)

        assert result["plan_id"] == "plan-1"
        call_args = client._session.request.call_args
        payload = call_args[1]["json"]
        assert payload["dry_run"] is True


class TestListNotebooks:
    def test_returns_notebooks(self) -> None:
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(
            200,
            {
                "notebooks": [
                    {
                        "notebook": "user_features",
                        "assets": [{"name": "fg.user_spend"}],
                    }
                ]
            },
        )

        notebooks = client.list_notebooks()
        assert len(notebooks) == 1
        assert notebooks[0]["notebook"] == "user_features"


class TestMarkDatasetReady:
    def test_returns_trigger_actions(self) -> None:
        client = _make_client()
        client._session = MagicMock()
        client._session.request.return_value = _mock_response(
            200,
            {
                "trigger_actions": [
                    {
                        "notebook": "user_features",
                        "trigger_type": "upstream",
                        "partition": "2025-01-15",
                    }
                ]
            },
        )

        actions = client.mark_dataset_ready(
            dataset_name="silver.orders",
            partition="2025-01-15",
            delta_version=42,
        )

        assert len(actions) == 1
        call_args = client._session.request.call_args
        payload = call_args[1]["json"]
        assert payload["dataset_name"] == "silver.orders"
        assert payload["delta_version"] == 42
