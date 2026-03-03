"""Tests for the CLI manifest generator."""
from __future__ import annotations

import json
import os
import textwrap

import pytest

from feature_compute_engine.cli.manifest import (
    build_manifest,
    diff_manifests,
    generate_manifest,
    read_manifest,
)
from feature_compute_engine.decorators.registry import AssetRegistry
from feature_compute_engine.types.asset_spec import AssetSpec, Input


@pytest.fixture(autouse=True)
def _clean_registry():
    AssetRegistry.clear()
    yield
    AssetRegistry.clear()


def _make_spec(name: str = "fg.test", **overrides) -> AssetSpec:
    defaults = dict(
        name=name,
        entity="user",
        entity_key="user_id",
        notebook="test_notebook",
        trigger="upstream",
    )
    defaults.update(overrides)
    return AssetSpec(**defaults)


def _write_notebook(tmp_path, filename: str, code: str) -> str:
    nb_dir = tmp_path / "notebooks"
    nb_dir.mkdir(exist_ok=True)
    filepath = nb_dir / filename
    filepath.write_text(textwrap.dedent(code))
    return str(nb_dir)


class TestBuildManifest:
    def test_produces_all_required_fields(self):
        specs = [_make_spec("fg.alpha"), _make_spec("fg.beta")]
        manifest = build_manifest(specs, git_sha="abc123", git_ref="main")

        assert manifest["version"] == "1"
        assert "generated_at" in manifest
        assert manifest["generated_by"] == "bharatml-cli"
        assert manifest["git_sha"] == "abc123"
        assert manifest["git_ref"] == "main"
        assert len(manifest["assets"]) == 2
        assert "summary" in manifest

    def test_assets_sorted_by_name(self):
        specs = [_make_spec("fg.zeta"), _make_spec("fg.alpha"), _make_spec("fg.mu")]
        manifest = build_manifest(specs)
        asset_names = [a["name"] for a in manifest["assets"]]
        assert asset_names == ["fg.alpha", "fg.mu", "fg.zeta"]

    def test_summary_counts(self):
        specs = [
            _make_spec("fg.a", entity="user", serving=True, schedule="3h"),
            _make_spec("fg.b", entity="user", serving=False, schedule="6h"),
            _make_spec(
                "fg.c",
                entity="product",
                entity_key="product_id",
                serving=True,
                schedule="3h",
            ),
        ]
        manifest = build_manifest(specs)
        summary = manifest["summary"]

        assert summary["total_assets"] == 3
        assert summary["by_entity"] == {"product": 1, "user": 2}
        assert summary["serving_count"] == 2
        assert summary["non_serving_count"] == 1

    def test_deterministic_same_input_same_output(self):
        specs = [_make_spec("fg.b"), _make_spec("fg.a")]
        m1 = build_manifest(specs, git_sha="abc", git_ref="main")
        m2 = build_manifest(specs, git_sha="abc", git_ref="main")

        # generated_at will differ, so compare everything else
        m1.pop("generated_at")
        m2.pop("generated_at")
        assert m1 == m2

    def test_assets_serialized_via_to_dict(self):
        spec = _make_spec(
            "fg.test",
            inputs=[Input("silver.orders", partition="ds", window=7)],
        )
        manifest = build_manifest([spec])
        asset_dict = manifest["assets"][0]
        assert asset_dict["name"] == "fg.test"
        assert len(asset_dict["inputs"]) == 1
        assert asset_dict["inputs"][0]["name"] == "silver.orders"
        assert asset_dict["inputs"][0]["window"] == 7


class TestGenerateManifest:
    def test_writes_manifest_file(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "features.py",
            """\
            from feature_compute_engine.decorators.asset import asset

            @asset(name="fg.test", entity="user", entity_key="user_id", schedule="3h")
            def test_fn(ctx, ds):
                pass
            """,
        )
        output_dir = str(tmp_path / "output")
        manifest = generate_manifest(
            notebook_dir=nb_dir,
            output_dir=output_dir,
            git_sha="sha123",
            git_ref="feature-branch",
        )

        manifest_path = os.path.join(output_dir, ".bharatml", "manifest.json")
        assert os.path.exists(manifest_path)

        with open(manifest_path, "r") as f:
            on_disk = json.load(f)
        assert on_disk["git_sha"] == "sha123"
        assert on_disk["summary"]["total_assets"] == 1

    def test_valid_json_output(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "features.py",
            """\
            from feature_compute_engine.decorators.asset import asset

            @asset(name="fg.x", entity="user", entity_key="user_id", schedule="3h")
            def x(ctx, ds):
                pass
            """,
        )
        output_dir = str(tmp_path / "output")
        generate_manifest(notebook_dir=nb_dir, output_dir=output_dir)

        manifest_path = os.path.join(output_dir, ".bharatml", "manifest.json")
        with open(manifest_path, "r") as f:
            data = json.load(f)
        assert isinstance(data, dict)
        assert isinstance(data["assets"], list)


class TestReadManifest:
    def test_read_roundtrip(self, tmp_path):
        manifest = {
            "version": "1",
            "assets": [{"name": "fg.test"}],
            "summary": {"total_assets": 1},
        }
        path = tmp_path / "manifest.json"
        with open(path, "w") as f:
            json.dump(manifest, f)

        loaded = read_manifest(str(path))
        assert loaded == manifest


class TestDiffManifests:
    def test_no_changes(self):
        manifest = {"assets": [{"name": "fg.a", "version": "v1", "inputs": []}]}
        diff = diff_manifests(manifest, manifest)
        assert diff["has_changes"] is False
        assert diff["added"] == []
        assert diff["removed"] == []
        assert diff["modified"] == []

    def test_added_asset_detected(self):
        current = {"assets": []}
        proposed = {"assets": [{"name": "fg.new", "version": "v1", "inputs": []}]}
        diff = diff_manifests(current, proposed)
        assert diff["has_changes"] is True
        assert diff["added"] == ["fg.new"]

    def test_removed_asset_detected(self):
        current = {"assets": [{"name": "fg.old", "version": "v1", "inputs": []}]}
        proposed = {"assets": []}
        diff = diff_manifests(current, proposed)
        assert diff["has_changes"] is True
        assert diff["removed"] == ["fg.old"]

    def test_modified_inputs_detected(self):
        current = {
            "assets": [
                {"name": "fg.a", "version": "v1", "inputs": [{"name": "silver.x"}]}
            ]
        }
        proposed = {
            "assets": [
                {"name": "fg.a", "version": "v1", "inputs": [{"name": "silver.y"}]}
            ]
        }
        diff = diff_manifests(current, proposed)
        assert diff["has_changes"] is True
        assert len(diff["modified"]) == 1
        assert "inputs_changed" in diff["modified"][0]["changes"]

    def test_modified_schedule_detected(self):
        current = {
            "assets": [
                {"name": "fg.a", "version": "v1", "schedule": "3h", "inputs": []}
            ]
        }
        proposed = {
            "assets": [
                {"name": "fg.a", "version": "v1", "schedule": "6h", "inputs": []}
            ]
        }
        diff = diff_manifests(current, proposed)
        assert diff["has_changes"] is True
        assert "schedule_changed" in diff["modified"][0]["changes"]

    def test_modified_serving_detected(self):
        current = {
            "assets": [
                {"name": "fg.a", "version": "v1", "serving": True, "inputs": []}
            ]
        }
        proposed = {
            "assets": [
                {"name": "fg.a", "version": "v1", "serving": False, "inputs": []}
            ]
        }
        diff = diff_manifests(current, proposed)
        assert diff["has_changes"] is True
        assert "serving_changed" in diff["modified"][0]["changes"]

    def test_code_changed_detected(self):
        current = {"assets": [{"name": "fg.a", "version": "v1", "inputs": []}]}
        proposed = {"assets": [{"name": "fg.a", "version": "v2", "inputs": []}]}
        diff = diff_manifests(current, proposed)
        assert diff["has_changes"] is True
        assert "code_changed" in diff["modified"][0]["changes"]

    def test_complex_diff(self):
        current = {
            "assets": [
                {"name": "fg.stays", "version": "v1", "inputs": []},
                {"name": "fg.removed", "version": "v1", "inputs": []},
                {"name": "fg.modified", "version": "v1", "schedule": "3h", "inputs": []},
            ]
        }
        proposed = {
            "assets": [
                {"name": "fg.stays", "version": "v1", "inputs": []},
                {"name": "fg.added", "version": "v1", "inputs": []},
                {"name": "fg.modified", "version": "v1", "schedule": "6h", "inputs": []},
            ]
        }
        diff = diff_manifests(current, proposed)
        assert diff["has_changes"] is True
        assert "fg.added" in diff["added"]
        assert "fg.removed" in diff["removed"]
        assert len(diff["modified"]) == 1
        assert diff["modified"][0]["name"] == "fg.modified"
