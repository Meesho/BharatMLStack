"""Tests for the AssetRegistry."""
from __future__ import annotations

import pytest

from feature_compute_engine.decorators.registry import AssetRegistry
from feature_compute_engine.types.asset_spec import AssetSpec, Input


@pytest.fixture(autouse=True)
def _clean_registry():
    """Clear the global registry before each test."""
    AssetRegistry.clear()
    yield
    AssetRegistry.clear()


def _make_spec(
    name: str = "fg.test",
    entity: str = "user",
    notebook: str = "user_notebook",
    **overrides,
) -> AssetSpec:
    defaults = dict(
        name=name,
        entity=entity,
        entity_key="user_id",
        notebook=notebook,
        trigger="upstream",
    )
    defaults.update(overrides)
    return AssetSpec(**defaults)


class TestAssetRegistry:
    def test_register_and_get(self) -> None:
        spec = _make_spec()
        AssetRegistry.register(spec)
        retrieved = AssetRegistry.get(spec.name)
        assert retrieved.name == spec.name
        assert retrieved.entity == spec.entity

    def test_get_nonexistent_raises(self) -> None:
        with pytest.raises(KeyError, match="not registered"):
            AssetRegistry.get("fg.nonexistent")

    def test_register_duplicate_same_notebook_updates(self) -> None:
        spec1 = _make_spec(serving=True)
        spec2 = _make_spec(serving=False)
        AssetRegistry.register(spec1)
        AssetRegistry.register(spec2)
        assert AssetRegistry.get("fg.test").serving is False
        assert AssetRegistry.count() == 1

    def test_register_duplicate_different_notebook_raises(self) -> None:
        spec1 = _make_spec(notebook="notebook_a")
        spec2 = _make_spec(notebook="notebook_b")
        AssetRegistry.register(spec1)
        with pytest.raises(ValueError, match="already registered in notebook"):
            AssetRegistry.register(spec2)

    def test_get_all(self) -> None:
        AssetRegistry.register(_make_spec(name="fg.a"))
        AssetRegistry.register(_make_spec(name="fg.b"))
        all_assets = AssetRegistry.get_all()
        assert len(all_assets) == 2
        assert "fg.a" in all_assets
        assert "fg.b" in all_assets

    def test_get_by_notebook(self) -> None:
        AssetRegistry.register(_make_spec(name="fg.a", notebook="nb1"))
        AssetRegistry.register(_make_spec(name="fg.b", notebook="nb2"))
        AssetRegistry.register(_make_spec(name="fg.c", notebook="nb1"))
        result = AssetRegistry.get_by_notebook("nb1")
        assert set(result.keys()) == {"fg.a", "fg.c"}

    def test_get_by_entity(self) -> None:
        AssetRegistry.register(_make_spec(name="fg.user1", entity="user"))
        AssetRegistry.register(
            _make_spec(
                name="fg.product1",
                entity="product",
                entity_key="product_id",
            )
        )
        AssetRegistry.register(_make_spec(name="fg.user2", entity="user"))
        result = AssetRegistry.get_by_entity("user")
        assert set(result.keys()) == {"fg.user1", "fg.user2"}

    def test_get_names(self) -> None:
        AssetRegistry.register(_make_spec(name="fg.x"))
        AssetRegistry.register(_make_spec(name="fg.y"))
        names = AssetRegistry.get_names()
        assert set(names) == {"fg.x", "fg.y"}

    def test_clear(self) -> None:
        AssetRegistry.register(_make_spec(name="fg.a"))
        AssetRegistry.register(_make_spec(name="fg.b"))
        assert AssetRegistry.count() == 2
        AssetRegistry.clear()
        assert AssetRegistry.count() == 0
        assert AssetRegistry.get_all() == {}

    def test_count(self) -> None:
        assert AssetRegistry.count() == 0
        AssetRegistry.register(_make_spec(name="fg.a"))
        assert AssetRegistry.count() == 1
        AssetRegistry.register(_make_spec(name="fg.b"))
        assert AssetRegistry.count() == 2

    def test_get_all_returns_copy(self) -> None:
        """Mutating the returned dict should not affect the registry."""
        AssetRegistry.register(_make_spec(name="fg.a"))
        all_assets = AssetRegistry.get_all()
        all_assets.clear()
        assert AssetRegistry.count() == 1
