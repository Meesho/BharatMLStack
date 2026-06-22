"""
No-Spark smoke test for the encoder + sharding logic.

Run:
    cd py-sdk/mnemosyne_producer
    pip install -e . --no-deps  # or just add the parent dir to PYTHONPATH
    PYTHONPATH=../bharatml_commons python examples/smoke_test.py

Asserts:
  * Config parses the sample geo_config.json.
  * RowEncoder turns a synthetic row into persist.Data bytes round-trippable
    via the existing persist_pb2 module.
  * shard_id is deterministic and in [0, num_shards).
  * Default values are applied when source columns are null.
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path


def main() -> int:
    here = Path(__file__).resolve().parent
    pkg_root = here.parent
    sys.path.insert(0, str(pkg_root))

    from mnemosyne_producer.config import Config
    from mnemosyne_producer.encoder import RowEncoder
    from mnemosyne_producer.sharding import make_key, shard_id

    cfg = Config.from_json(str(here / "geo_config.json"))
    assert cfg.entity_label == "catalog__user_geohash_1_3"
    assert cfg.key_columns == ["geohash_1_3_id", "catalog_id"]
    fg_labels = [fg.label for fg in cfg.feature_groups]
    assert "derived_fp32" in fg_labels
    assert "rollup_int32" in fg_labels
    print(f"config OK: entity={cfg.entity_label} keys={cfg.key_columns} "
          f"fgs={fg_labels} features_total={sum(len(fg.features) for fg in cfg.feature_groups)}")

    # Synthetic row with a couple of features filled and most as nulls (defaults apply).
    row = {
        "geohash_1_3_id": "abc123",
        "catalog_id": 99,
        # one filled FP32 source col:
        "user_geohash_res_1__orders_by_clicks_7_days_ewma": 0.42,
        # one filled Int32 source col:
        "user_geohash_res_3__clicks_3_days": 7,
        # the rest are None -> defaults from config (= 0)
    }
    # Sharding + key
    key_str = make_key(cfg.key_columns, row)
    assert key_str == "abc123|99", key_str
    sid = shard_id(key_str, 10)
    assert 0 <= sid < 10
    print(f"key={key_str!r} shard_id={sid}/10")

    # Encode
    encoder = RowEncoder(cfg)
    blob = encoder.encode(row)
    assert isinstance(blob, bytes) and len(blob) > 0

    # Decode & verify round-trip
    from bharatml_commons.proto.persist.persist_pb2 import Data
    data = Data()
    data.ParseFromString(blob)
    assert list(data.key_values) == ["abc123", "99"]
    assert len(data.feature_values) == len(cfg.feature_groups)

    # Find derived_fp32 block by position
    fg_idx = {fg.label: i for i, fg in enumerate(cfg.feature_groups)}
    fp32_block = data.feature_values[fg_idx["derived_fp32"]].values
    int32_block = data.feature_values[fg_idx["rollup_int32"]].values
    print(f"derived_fp32: {len(fp32_block.fp32_values)} fp32 values "
          f"(first 3 = {list(fp32_block.fp32_values[:3])})")
    print(f"rollup_int32: {len(int32_block.int32_values)} int32 values "
          f"(values = {list(int32_block.int32_values)})")

    # The filled features should show non-default values:
    derived = next(fg for fg in cfg.feature_groups if fg.label == "derived_fp32")
    filled_idx = [i for i, f in enumerate(derived.features)
                  if f.source_column == "user_geohash_res_1__orders_by_clicks_7_days_ewma"][0]
    assert abs(fp32_block.fp32_values[filled_idx] - 0.42) < 1e-6

    rollup = next(fg for fg in cfg.feature_groups if fg.label == "rollup_int32")
    filled_idx2 = [i for i, f in enumerate(rollup.features)
                   if f.source_column == "user_geohash_res_3__clicks_3_days"][0]
    assert int32_block.int32_values[filled_idx2] == 7

    print("ALL GOOD")
    return 0


if __name__ == "__main__":
    sys.exit(main())
