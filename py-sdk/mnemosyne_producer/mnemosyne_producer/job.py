"""
Mnemosyne Producer — PySpark batch job.

Pipeline
========

  1. Read input parquet (one or many files).
  2. Project to only the columns needed (key cols + feature source cols).
  3. Build the composite key   `__mnemo_key  = concat_ws('|', key_cols...)`.
  4. Compute the shard         `__mnemo_shard = crc32(__mnemo_key) % S`
     using Spark SQL's `crc32()` (IEEE polynomial — identical to
     `binascii.crc32` and Go's `hash/crc32.ChecksumIEEE`).
  5. Repartition by `__mnemo_shard` and `sortWithinPartitions` by
     (`__mnemo_shard`, `__mnemo_key`).
  6. `mapPartitions` — encode each row to a persist.Data proto and write
     SST files via `rocksdict.SstFileWriter` (sorted insert).
  7. Collect per-shard SST manifests and emit a global `_manifest.json`
     for the version.

Output layout
=============

  <output>/<version_id>/<shard:05d>/data-<partition_uuid>.sst
  <output>/<version_id>/_manifest.json

Submit (local)
==============

  spark-submit \
      --packages "io.delta:delta-core_2.12:2.4.0" \
      -m mnemosyne_producer.job \
      --input  /path/to/*.snappy.parquet \
      --config /path/to/geo_config.json \
      --output /tmp/mnemo_out                \
      --num-shards 10 \
      --version-id 20260622_r1
"""
from __future__ import annotations

import argparse
import json
import os
import sys
from typing import Any, Dict, List

from .config import Config
from .sharding import KEY_SEP


_SHARD_COL = "__mnemo_shard"
_KEY_COL = "__mnemo_key"


def parse_args(argv: List[str] | None = None) -> argparse.Namespace:
    ap = argparse.ArgumentParser(description="Mnemosyne Producer (PySpark)")
    ap.add_argument("--input", required=True,
                    help="Parquet input — file, directory, or glob (Spark-readable)")
    ap.add_argument("--config", required=True,
                    help="Path to onboarding JSON config (entity/keys/features)")
    ap.add_argument("--output", required=True,
                    help="Base output dir (local path or DBFS-mounted path)")
    ap.add_argument("--num-shards", type=int, required=True)
    ap.add_argument("--version-id", required=True,
                    help="Version identifier, e.g. '20260622_r1'")
    ap.add_argument("--shuffle-partitions", type=int, default=None,
                    help="Override spark.sql.shuffle.partitions (default: leave as is)")
    ap.add_argument("--app-name", default="mnemosyne-producer")
    return ap.parse_args(argv)


def build_spark(app_name: str, shuffle_partitions: int | None):
    from pyspark.sql import SparkSession
    builder = SparkSession.builder.appName(app_name)
    if shuffle_partitions is not None:
        builder = builder.config("spark.sql.shuffle.partitions", str(shuffle_partitions))
    return builder.getOrCreate()


def run(args: argparse.Namespace) -> None:
    from pyspark.sql import functions as F
    from pyspark.sql.types import LongType

    cfg = Config.from_json(args.config)
    spark = build_spark(args.app_name, args.shuffle_partitions)

    # ----- 1. Read & project -------------------------------------------------
    df = spark.read.parquet(args.input)
    project_cols = cfg.all_source_columns()
    missing = [c for c in project_cols if c not in df.columns]
    if missing:
        raise ValueError(
            f"Input parquet is missing required columns: {missing[:10]}"
            f"{' ...' if len(missing) > 10 else ''}"
        )
    df = df.select(*project_cols)

    # ----- 2. Composite key + shard_id (vectorised in Spark SQL) -------------
    key_parts = [
        F.coalesce(F.col(c).cast("string"), F.lit("")) for c in cfg.key_columns
    ]
    df = df.withColumn(_KEY_COL, F.concat_ws(KEY_SEP, *key_parts))
    df = df.withColumn(
        _SHARD_COL,
        (F.crc32(F.col(_KEY_COL)) % F.lit(args.num_shards)).cast("int"),
    )

    # ----- 3. Repartition by shard, sort within partition --------------------
    df = (
        df.repartition(args.num_shards, F.col(_SHARD_COL))
          .sortWithinPartitions(_SHARD_COL, _KEY_COL)
    )

    # ----- 4. Encode + write SSTs in mapPartitions ---------------------------
    out_base = args.output.rstrip("/")
    version_id = args.version_id
    cfg_dict_for_workers = cfg.schema_dict()  # cheap, picklable

    def _process(rows_iter):
        # Imports happen on the executor, once per partition.
        from mnemosyne_producer.encoder import RowEncoder
        from mnemosyne_producer.config import Config as _Config
        from mnemosyne_producer.sst_writer import write_partition_ssts
        cfg_local = _Config(
            entity_label=cfg_dict_for_workers["entity_label"],
            key_columns=cfg_dict_for_workers["key_columns"],
            feature_groups=[
                _make_fg(fg_dict) for fg_dict in cfg_dict_for_workers["feature_groups"]
            ],
            source_base_paths=cfg_dict_for_workers.get("source_base_paths", []),
        )
        encoder = RowEncoder(cfg_local)
        yield from write_partition_ssts(rows_iter, encoder, out_base, version_id)

    manifests: List[Dict[str, Any]] = df.rdd.mapPartitions(_process).collect()

    # ----- 5. Global manifest ------------------------------------------------
    by_shard: Dict[int, List[Dict[str, Any]]] = {}
    for m in manifests:
        by_shard.setdefault(int(m["shard_id"]), []).append(m)

    global_manifest = {
        "version_id": version_id,
        "num_shards": args.num_shards,
        "key_separator": KEY_SEP,
        "value_format": {
            "encoding": "protobuf",
            "schema": "persist.Data (BharatMLStack/online-feature-store/pkg/proto/persist.proto)",
            "fg_order": [fg["label"] for fg in cfg_dict_for_workers["feature_groups"]],
        },
        "schema": cfg_dict_for_workers,
        "shards": {
            str(sid): {
                "row_count": sum(m["rows"] for m in entries),
                "files": [
                    {
                        "path": os.path.relpath(m["path"], out_base),
                        "rows": m["rows"],
                        "bytes_logical": m["bytes_logical"],
                        "size_on_disk": m["size_on_disk"],
                        "sha256": m["sha256"],
                    }
                    for m in entries
                ],
            }
            for sid, entries in sorted(by_shard.items())
        },
        "stats": {
            "total_rows": sum(m["rows"] for m in manifests),
            "total_files": len(manifests),
            "shards_with_data": len(by_shard),
        },
    }

    manifest_dir = os.path.join(out_base, version_id)
    os.makedirs(manifest_dir, exist_ok=True)
    manifest_path = os.path.join(manifest_dir, "_manifest.json")
    with open(manifest_path, "w") as f:
        json.dump(global_manifest, f, indent=2, default=str)

    print(f"[mnemosyne-producer] wrote {len(manifests)} SST files across "
          f"{len(by_shard)} / {args.num_shards} shards "
          f"({global_manifest['stats']['total_rows']:,} rows total)")
    print(f"[mnemosyne-producer] global manifest: {manifest_path}")


def _make_fg(d: Dict[str, Any]):
    """Rebuild a FeatureGroupSpec from its schema dict (executor-side)."""
    from .config import FeatureGroupSpec, FeatureSpec
    return FeatureGroupSpec(
        label=d["label"],
        features=[
            FeatureSpec(
                label=f["label"],
                source_column=f["source_column"],
                default_value=f["default_value"],
                data_type=f["data_type"],
            )
            for f in d["features"]
        ],
    )


def main(argv: List[str] | None = None) -> int:
    args = parse_args(argv)
    run(args)
    return 0


if __name__ == "__main__":
    sys.exit(main())
