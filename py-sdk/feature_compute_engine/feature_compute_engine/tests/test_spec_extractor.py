"""Tests for the CLI spec extractor."""
from __future__ import annotations

import os
import textwrap

import pytest

from feature_compute_engine.cli.spec_extractor import ExtractionError, extract_specs
from feature_compute_engine.decorators.registry import AssetRegistry


@pytest.fixture(autouse=True)
def _clean_registry():
    AssetRegistry.clear()
    yield
    AssetRegistry.clear()


def _write_notebook(tmp_path, filename: str, code: str) -> str:
    """Write a .py file in the temp directory and return the directory path."""
    nb_dir = tmp_path / "notebooks"
    nb_dir.mkdir(exist_ok=True)
    filepath = nb_dir / filename
    filepath.write_text(textwrap.dedent(code))
    return str(nb_dir)


class TestExtractSpecs:
    def test_extracts_asset_from_decorated_function(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "user_features.py",
            """\
            from feature_compute_engine.decorators.asset import asset
            from feature_compute_engine.types.asset_spec import Input

            @asset(
                name="fg.user_spend",
                entity="user",
                entity_key="user_id",
                schedule="3h",
                serving=True,
                inputs=[Input("silver.orders")],
            )
            def user_spend(ctx, ds, orders_df):
                pass
            """,
        )
        specs = extract_specs(nb_dir)
        assert len(specs) == 1
        assert specs[0].name == "fg.user_spend"
        assert specs[0].entity == "user"
        assert specs[0].entity_key == "user_id"
        assert specs[0].serving is True
        assert len(specs[0].inputs) == 1
        assert specs[0].inputs[0].name == "silver.orders"

    def test_extracts_multiple_assets_from_single_file(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "user_features.py",
            """\
            from feature_compute_engine.decorators.asset import asset
            from feature_compute_engine.types.asset_spec import Input

            @asset(
                name="fg.user_spend",
                entity="user",
                entity_key="user_id",
                schedule="3h",
            )
            def user_spend(ctx, ds):
                pass

            @asset(
                name="fg.user_orders",
                entity="user",
                entity_key="user_id",
                trigger="upstream",
                inputs=[Input("silver.orders")],
            )
            def user_orders(ctx, ds, orders_df):
                pass
            """,
        )
        specs = extract_specs(nb_dir)
        assert len(specs) == 2
        names = {s.name for s in specs}
        assert names == {"fg.user_spend", "fg.user_orders"}

    def test_file_with_no_decorators_returns_empty(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "utils.py",
            """\
            def helper():
                return 42
            """,
        )
        specs = extract_specs(nb_dir)
        assert len(specs) == 0

    def test_syntax_error_raises_extraction_error(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "broken.py",
            """\
            def broken(
                # missing closing paren
            """,
        )
        with pytest.raises(ExtractionError) as exc_info:
            extract_specs(nb_dir, fail_on_error=True)
        assert "broken.py" in str(exc_info.value)

    def test_syntax_error_skipped_when_fail_on_error_false(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "broken.py",
            """\
            def broken(
            """,
        )
        specs = extract_specs(nb_dir, fail_on_error=False)
        assert len(specs) == 0

    def test_git_sha_stamped_on_all_specs(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "user_features.py",
            """\
            from feature_compute_engine.decorators.asset import asset

            @asset(name="fg.a", entity="user", entity_key="user_id", schedule="3h")
            def asset_a(ctx, ds):
                pass

            @asset(name="fg.b", entity="user", entity_key="user_id", schedule="6h")
            def asset_b(ctx, ds):
                pass
            """,
        )
        sha = "abc123def456"
        specs = extract_specs(nb_dir, git_sha=sha)
        assert len(specs) == 2
        for spec in specs:
            assert spec.version == sha

    def test_notebook_name_normalization(self, tmp_path):
        nb_dir = _write_notebook(
            tmp_path,
            "user_features.py",
            """\
            from feature_compute_engine.decorators.asset import asset

            @asset(name="fg.test", entity="user", entity_key="user_id", schedule="3h")
            def test_asset(ctx, ds):
                pass
            """,
        )
        specs = extract_specs(nb_dir)
        assert len(specs) == 1
        assert not specs[0].notebook.startswith("_bharatml_extract_.")

    def test_nonexistent_directory_raises_file_not_found(self, tmp_path):
        with pytest.raises(FileNotFoundError, match="not found"):
            extract_specs(str(tmp_path / "nonexistent"))

    def test_empty_directory_returns_empty_list(self, tmp_path):
        nb_dir = tmp_path / "notebooks"
        nb_dir.mkdir()
        specs = extract_specs(str(nb_dir))
        assert specs == []

    def test_underscore_files_are_skipped(self, tmp_path):
        nb_dir = tmp_path / "notebooks"
        nb_dir.mkdir()
        (nb_dir / "__init__.py").write_text("# init")
        (nb_dir / "_private.py").write_text("x = 1")
        (nb_dir / "real.py").write_text(
            textwrap.dedent("""\
            from feature_compute_engine.decorators.asset import asset

            @asset(name="fg.real", entity="user", entity_key="user_id", schedule="3h")
            def real_asset(ctx, ds):
                pass
            """)
        )
        specs = extract_specs(str(nb_dir))
        assert len(specs) == 1
        assert specs[0].name == "fg.real"

    def test_multiple_files_across_directories(self, tmp_path):
        nb_dir = tmp_path / "notebooks"
        nb_dir.mkdir()
        sub = nb_dir / "subdir"
        sub.mkdir()

        (nb_dir / "user.py").write_text(
            textwrap.dedent("""\
            from feature_compute_engine.decorators.asset import asset

            @asset(name="fg.user1", entity="user", entity_key="user_id", schedule="3h")
            def user_asset(ctx, ds):
                pass
            """)
        )
        (sub / "product.py").write_text(
            textwrap.dedent("""\
            from feature_compute_engine.decorators.asset import asset

            @asset(name="fg.product1", entity="product", entity_key="product_id", schedule="6h")
            def product_asset(ctx, ds):
                pass
            """)
        )

        specs = extract_specs(str(nb_dir))
        assert len(specs) == 2
        names = {s.name for s in specs}
        assert names == {"fg.user1", "fg.product1"}
