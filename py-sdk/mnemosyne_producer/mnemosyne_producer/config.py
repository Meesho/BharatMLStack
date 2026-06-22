"""
Parse the onboarding JSON config into a typed structure.

The input JSON looks like:

    {
      "data": [
        {
          "storage-provider": "TABLE",
          "base-path": [
            {
              "source-base-path": "ds_dbc_ofs.catalog__user_geohash_1_3__derived",
              "data-paths": [
                { "entity-label": "catalog__user_geohash_1_3",
                  "feature-group-label": "derived_fp32",
                  "feature-label": "orders_by_clicks_7_days_ewma",
                  "source-data-column": "user_geohash_res_1__orders_by_clicks_7_days_ewma",
                  "default-value": "0",
                  "data-type": "DataTypeFP32" },
                ...
              ]
            }
          ]
        }
      ],
      "keys": { "catalog__user_geohash_1_3": ["geohash_1_3_id", "catalog_id"] }
    }

Constraints / V1 scope:
  * Single entity_label per config (multi-entity later — one config per store).
  * All features within one feature-group-label share the same data-type.
  * Feature-group + feature order is preserved from the JSON (insertion order).
    Producer and consumer must agree on this order; we emit the schema to the
    output manifest so the consumer can decode positionally.
"""
from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Any, Dict, List


@dataclass(frozen=True)
class FeatureSpec:
    label: str            # e.g., "orders_by_clicks_7_days_ewma"
    source_column: str    # e.g., "user_geohash_res_1__orders_by_clicks_7_days_ewma"
    default_value: str    # raw default from JSON; cast at encode time
    data_type: str        # e.g., "DataTypeFP32"


@dataclass
class FeatureGroupSpec:
    label: str
    features: List[FeatureSpec] = field(default_factory=list)

    @property
    def data_type(self) -> str:
        if not self.features:
            return ""
        return self.features[0].data_type

    def assert_uniform_dtype(self) -> None:
        dt = self.data_type
        for f in self.features:
            if f.data_type != dt:
                raise ValueError(
                    f"feature-group '{self.label}' has mixed data-types: "
                    f"expected {dt}, got {f.data_type} on feature {f.label!r}"
                )


@dataclass
class Config:
    entity_label: str
    key_columns: List[str]
    feature_groups: List[FeatureGroupSpec]
    # The original source-base-path declared in the JSON (informational; the actual
    # input is supplied at job submit time via --input).
    source_base_paths: List[str] = field(default_factory=list)

    def all_source_columns(self) -> List[str]:
        cols: List[str] = []
        seen = set()
        for c in self.key_columns:
            if c not in seen:
                cols.append(c); seen.add(c)
        for fg in self.feature_groups:
            for f in fg.features:
                if f.source_column not in seen:
                    cols.append(f.source_column); seen.add(f.source_column)
        return cols

    def schema_dict(self) -> Dict[str, Any]:
        """Schema written into the global manifest so consumers can decode positionally."""
        return {
            "entity_label": self.entity_label,
            "key_columns": list(self.key_columns),
            "feature_groups": [
                {
                    "label": fg.label,
                    "data_type": fg.data_type,
                    "features": [
                        {
                            "label": f.label,
                            "source_column": f.source_column,
                            "default_value": f.default_value,
                            "data_type": f.data_type,
                        }
                        for f in fg.features
                    ],
                }
                for fg in self.feature_groups
            ],
            "source_base_paths": list(self.source_base_paths),
        }

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "Config":
        keys = d.get("keys") or {}
        if len(keys) != 1:
            raise ValueError("Config 'keys' must contain exactly one entity_label (V1).")
        entity_label = next(iter(keys))
        key_columns = list(keys[entity_label])

        fg_map: Dict[str, FeatureGroupSpec] = {}
        source_paths: List[str] = []

        for block in d.get("data", []):
            for base in block.get("base-path", []) or []:
                source_paths.append(base.get("source-base-path", ""))
                for dp in base.get("data-paths", []) or []:
                    if dp.get("entity-label") != entity_label:
                        # Skip rows that belong to a different entity (defensive).
                        continue
                    fg_label = dp["feature-group-label"]
                    fg = fg_map.setdefault(fg_label, FeatureGroupSpec(label=fg_label))
                    fg.features.append(FeatureSpec(
                        label=dp["feature-label"],
                        source_column=dp["source-data-column"],
                        default_value=str(dp.get("default-value", "0")),
                        data_type=dp["data-type"],
                    ))

        feature_groups = list(fg_map.values())
        for fg in feature_groups:
            fg.assert_uniform_dtype()

        return cls(
            entity_label=entity_label,
            key_columns=key_columns,
            feature_groups=feature_groups,
            source_base_paths=source_paths,
        )

    @classmethod
    def from_json(cls, path: str) -> "Config":
        with open(path) as f:
            return cls.from_dict(json.load(f))
