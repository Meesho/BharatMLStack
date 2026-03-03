"""Tests for the bharatml CLI main entry point."""
from __future__ import annotations

import json
import os
import textwrap
from unittest.mock import patch

import pytest

from feature_compute_engine.cli.main import main, _build_parser
from feature_compute_engine.decorators.registry import AssetRegistry


@pytest.fixture(autouse=True)
def _clean_registry():
    AssetRegistry.clear()
    yield
    AssetRegistry.clear()


def _write_notebook(tmp_path, filename: str, code: str) -> str:
    nb_dir = tmp_path / "notebooks"
    nb_dir.mkdir(exist_ok=True)
    (nb_dir / filename).write_text(textwrap.dedent(code))
    return str(nb_dir)


class TestParser:
    def test_manifest_subcommand_defaults(self):
        parser = _build_parser()
        args = parser.parse_args(["manifest"])
        assert args.command == "manifest"
        assert args.notebook_dir == "./notebooks"
        assert args.output_dir == "."
        assert args.git_sha is None

    def test_diff_subcommand(self):
        parser = _build_parser()
        args = parser.parse_args(["diff", "--base", "a.json", "--proposed", "b.json"])
        assert args.command == "diff"
        assert args.base == "a.json"
        assert args.proposed == "b.json"

    def test_apply_requires_horizon_url(self):
        parser = _build_parser()
        with pytest.raises(SystemExit):
            parser.parse_args(["apply"])

    def test_no_command_returns_none(self):
        parser = _build_parser()
        args = parser.parse_args([])
        assert args.command is None


class TestManifestCommand:
    def test_manifest_generates_file(self, tmp_path, capsys):
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
        main([
            "manifest",
            "--notebook-dir", nb_dir,
            "--output-dir", output_dir,
            "--git-sha", "testsha",
        ])

        manifest_path = os.path.join(output_dir, ".bharatml", "manifest.json")
        assert os.path.exists(manifest_path)

        with open(manifest_path) as f:
            data = json.load(f)
        assert data["summary"]["total_assets"] == 1

        captured = capsys.readouterr()
        assert "1 assets" in captured.out


class TestDiffCommand:
    def test_diff_no_changes(self, tmp_path, capsys):
        manifest = {"assets": [{"name": "fg.a", "version": "v1", "inputs": []}]}
        base_path = tmp_path / "base.json"
        proposed_path = tmp_path / "proposed.json"
        with open(base_path, "w") as f:
            json.dump(manifest, f)
        with open(proposed_path, "w") as f:
            json.dump(manifest, f)

        main([
            "diff",
            "--base", str(base_path),
            "--proposed", str(proposed_path),
        ])

        captured = capsys.readouterr()
        assert "No changes" in captured.out

    def test_diff_shows_added(self, tmp_path, capsys):
        base = {"assets": []}
        proposed = {"assets": [{"name": "fg.new", "version": "v1", "inputs": []}]}
        base_path = tmp_path / "base.json"
        proposed_path = tmp_path / "proposed.json"
        with open(base_path, "w") as f:
            json.dump(base, f)
        with open(proposed_path, "w") as f:
            json.dump(proposed, f)

        main([
            "diff",
            "--base", str(base_path),
            "--proposed", str(proposed_path),
        ])

        captured = capsys.readouterr()
        assert "NEW" in captured.out
        assert "fg.new" in captured.out

    def test_diff_json_format(self, tmp_path, capsys):
        base = {"assets": []}
        proposed = {"assets": [{"name": "fg.new", "version": "v1", "inputs": []}]}
        base_path = tmp_path / "base.json"
        proposed_path = tmp_path / "proposed.json"
        with open(base_path, "w") as f:
            json.dump(base, f)
        with open(proposed_path, "w") as f:
            json.dump(proposed, f)

        main([
            "diff",
            "--base", str(base_path),
            "--proposed", str(proposed_path),
            "--format", "json",
        ])

        captured = capsys.readouterr()
        result = json.loads(captured.out)
        assert result["has_changes"] is True
        assert "fg.new" in result["added"]


class TestNoCommand:
    def test_no_command_exits(self):
        with pytest.raises(SystemExit) as exc_info:
            main([])
        assert exc_info.value.code == 1
