"""Tests for result types."""
from __future__ import annotations

from feature_compute_engine.runtime.result import AssetResult, NotebookRunResult


class TestAssetResult:
    def test_succeeded(self) -> None:
        r = AssetResult(asset_name="fg.a", action="executed")
        assert r.succeeded is True
        assert r.skipped is False
        assert r.failed is False

    def test_skipped_cached(self) -> None:
        r = AssetResult(asset_name="fg.a", action="skipped_cached")
        assert r.succeeded is False
        assert r.skipped is True
        assert r.failed is False

    def test_skipped_unnecessary(self) -> None:
        r = AssetResult(asset_name="fg.a", action="skipped_unnecessary")
        assert r.skipped is True

    def test_failed(self) -> None:
        r = AssetResult(asset_name="fg.a", action="failed", error="boom")
        assert r.failed is True
        assert r.succeeded is False
        assert r.skipped is False

    def test_to_dict(self) -> None:
        r = AssetResult(
            asset_name="fg.a",
            action="executed",
            compute_key="key1",
            row_count=100,
            dq_results={"row_count > 0": True},
        )
        d = r.to_dict()
        assert d["asset_name"] == "fg.a"
        assert d["action"] == "executed"
        assert d["compute_key"] == "key1"
        assert d["row_count"] == 100
        assert d["dq_results"] == {"row_count > 0": True}


class TestNotebookRunResult:
    def _make_results(self) -> list:
        return [
            AssetResult(asset_name="fg.a", action="executed"),
            AssetResult(asset_name="fg.b", action="skipped_cached"),
            AssetResult(asset_name="fg.c", action="failed", error="err"),
        ]

    def test_counts(self) -> None:
        rr = NotebookRunResult(
            notebook="nb1",
            trigger_type="schedule_3h",
            partition="2026-03-01",
            results=self._make_results(),
        )
        assert rr.executed_count == 1
        assert rr.skipped_count == 1
        assert rr.failed_count == 1

    def test_all_succeeded_false_when_failures(self) -> None:
        rr = NotebookRunResult(
            notebook="nb1",
            trigger_type="schedule_3h",
            partition="2026-03-01",
            results=self._make_results(),
        )
        assert rr.all_succeeded is False

    def test_all_succeeded_true(self) -> None:
        rr = NotebookRunResult(
            notebook="nb1",
            trigger_type="schedule_3h",
            partition="2026-03-01",
            results=[
                AssetResult(asset_name="fg.a", action="executed"),
                AssetResult(asset_name="fg.b", action="skipped_cached"),
            ],
        )
        assert rr.all_succeeded is True

    def test_to_dict(self) -> None:
        rr = NotebookRunResult(
            notebook="nb1",
            trigger_type="schedule_3h",
            partition="2026-03-01",
            results=[AssetResult(asset_name="fg.a", action="executed")],
        )
        d = rr.to_dict()
        assert d["notebook"] == "nb1"
        assert d["summary"]["executed"] == 1
        assert d["summary"]["failed"] == 0
        assert d["summary"]["skipped"] == 0
        assert len(d["results"]) == 1
