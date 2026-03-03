"""Tests for AssetSpec and Input dataclasses."""
from __future__ import annotations

import json

import pytest

from feature_compute_engine.types.asset_spec import AssetSpec, Input


class TestInput:
    def test_valid_input(self) -> None:
        inp = Input(name="silver.orders", partition="ds", window=7)
        assert inp.name == "silver.orders"
        assert inp.partition == "ds"
        assert inp.window == 7
        assert inp.input_type == "internal"

    def test_defaults(self) -> None:
        inp = Input(name="silver.orders")
        assert inp.partition == "ds"
        assert inp.window is None
        assert inp.input_type == "internal"

    def test_external_input(self) -> None:
        inp = Input(name="raw.events", input_type="external")
        assert inp.input_type == "external"

    def test_empty_name_raises(self) -> None:
        with pytest.raises(ValueError, match="Input name cannot be empty"):
            Input(name="")

    def test_negative_window_raises(self) -> None:
        with pytest.raises(ValueError, match="Window size must be >= 1"):
            Input(name="silver.orders", window=0)

    def test_window_minus_one_raises(self) -> None:
        with pytest.raises(ValueError, match="Window size must be >= 1"):
            Input(name="silver.orders", window=-1)

    def test_frozen(self) -> None:
        inp = Input(name="silver.orders")
        with pytest.raises(AttributeError):
            inp.name = "other"  # type: ignore[misc]


class TestAssetSpec:
    def _make_spec(self, **overrides) -> AssetSpec:
        defaults = dict(
            name="fg.user_spend",
            entity="user",
            entity_key="user_id",
            notebook="user_notebook",
            schedule="3h",
            inputs=[
                Input("silver.orders", partition="ds"),
                Input("fg.base_features", partition="ds"),
            ],
        )
        defaults.update(overrides)
        return AssetSpec(**defaults)

    def test_valid_creation(self) -> None:
        spec = self._make_spec()
        assert spec.name == "fg.user_spend"
        assert spec.entity == "user"
        assert spec.entity_key == "user_id"
        assert spec.serving is True
        assert spec.incremental is False

    def test_trigger_normalization(self) -> None:
        spec = self._make_spec(schedule="3h", trigger="upstream")
        assert spec.trigger == "schedule_3h"

    def test_trigger_already_schedule_prefix(self) -> None:
        spec = self._make_spec(schedule="6h", trigger="schedule_6h")
        assert spec.trigger == "schedule_6h"

    def test_upstream_trigger_no_schedule(self) -> None:
        spec = self._make_spec(trigger="upstream", schedule=None)
        assert spec.trigger == "upstream"
        assert spec.schedule is None

    def test_empty_name_raises(self) -> None:
        with pytest.raises(ValueError, match="Asset name cannot be empty"):
            self._make_spec(name="")

    def test_empty_entity_raises(self) -> None:
        with pytest.raises(ValueError, match="Entity cannot be empty"):
            self._make_spec(entity="")

    def test_empty_entity_key_raises(self) -> None:
        with pytest.raises(ValueError, match="Entity key cannot be empty"):
            self._make_spec(entity_key="")

    def test_schedule_trigger_without_schedule_raises(self) -> None:
        with pytest.raises(ValueError, match="Schedule is required"):
            AssetSpec(
                name="fg.test",
                entity="user",
                entity_key="user_id",
                notebook="nb",
                trigger="schedule_3h",
                schedule=None,
            )

    def test_input_names(self) -> None:
        spec = self._make_spec()
        assert spec.input_names == ["silver.orders", "fg.base_features"]

    def test_internal_external_inputs(self) -> None:
        spec = self._make_spec(
            inputs=[
                Input("silver.orders", input_type="external"),
                Input("fg.base", input_type="internal"),
            ]
        )
        assert len(spec.internal_inputs) == 1
        assert spec.internal_inputs[0].name == "fg.base"
        assert len(spec.external_inputs) == 1
        assert spec.external_inputs[0].name == "silver.orders"

    def test_serialization_roundtrip(self) -> None:
        original = self._make_spec(
            freshness="2h",
            checks=["row_count > 0"],
            version="abc123",
        )
        d = original.to_dict()
        restored = AssetSpec.from_dict(d)

        assert restored.name == original.name
        assert restored.entity == original.entity
        assert restored.entity_key == original.entity_key
        assert restored.notebook == original.notebook
        assert restored.trigger == original.trigger
        assert restored.schedule == original.schedule
        assert restored.serving == original.serving
        assert restored.incremental == original.incremental
        assert restored.freshness == original.freshness
        assert restored.checks == original.checks
        assert restored.version == original.version
        assert len(restored.inputs) == len(original.inputs)
        for r, o in zip(restored.inputs, original.inputs):
            assert r.name == o.name
            assert r.partition == o.partition
            assert r.window == o.window
            assert r.input_type == o.input_type

    def test_to_json_is_valid_json(self) -> None:
        spec = self._make_spec()
        raw = spec.to_json()
        parsed = json.loads(raw)
        assert parsed["name"] == "fg.user_spend"

    def test_compute_fn_excluded_from_dict(self) -> None:
        fn = lambda ctx, ds: None  # noqa: E731
        spec = self._make_spec(_compute_fn=fn)
        d = spec.to_dict()
        assert "_compute_fn" not in d
        assert "compute_fn" not in d

    def test_from_dict_defaults(self) -> None:
        minimal = {
            "name": "fg.test",
            "entity": "user",
            "entity_key": "user_id",
            "notebook": "nb",
        }
        spec = AssetSpec.from_dict(minimal)
        assert spec.partition == "ds"
        assert spec.trigger == "upstream"
        assert spec.serving is True
        assert spec.incremental is False
        assert spec.inputs == []
        assert spec.checks == []
