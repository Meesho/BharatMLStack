# `mnemosyne_producer`

PySpark batch job — the **Producer** layer of Mnemosyne. Reads parquet, shards
by `crc32(key) % S`, encodes per-entity payloads as `persist.Data` protobuf,
and writes per-shard sorted RocksDB SST files plus a version manifest.

```
parquet (N files)  ──►  Spark SQL: project + key + shard
                            │
                            ▼  repartition by shard, sort within partition
                       mapPartitions
                            │
                            ▼
        ┌──────────────────────────────────────────┐
        │  per-Spark-partition:                    │
        │   RowEncoder → persist.Data → bytes      │
        │   rocksdict.SstFileWriter (sorted insert)│
        └──────────────────────────────────────────┘
                            │
                            ▼
   <output>/<version_id>/<shard:05d>/data-<partition_uuid>.sst
   <output>/<version_id>/_manifest.json
```

## Key design choices

| Decision | Choice | Why |
|---|---|---|
| RocksDB key | `concat_ws('\|', key_cols)` UTF-8 | matches existing `getKeyString` convention; readable; deterministic |
| RocksDB value | one entry per **entity**, value = `persist.Data` proto bytes | single point-lookup serves all features for an entity (optimal WORM read) |
| Shard hash | `crc32(IEEE) % S` | matches Spark SQL `crc32()`, Python `binascii.crc32`, Go `crc32.ChecksumIEEE` |
| Compression | none at SST level (RocksDB block compression is engine-side) | keep producer simple; serving engine picks |
| Sorted ingest | `sortWithinPartitions("__mnemo_shard", "__mnemo_key")` | required by `SstFileWriter.put` (sorted-insert contract) |
| SSTs per shard | possibly multiple (one per partition that holds rows for that shard) | RocksDB `IngestExternalFile` handles overlapping SSTs |

## Inputs

* **Config** (the onboarding JSON, see `examples/geo_config.json`) — declares the
  entity, key columns, feature-group/feature/source-column/default-value/data-type.
* **Input parquet** — single path, directory, or glob. All files must contain
  every column referenced by the config (key columns + feature source columns).
* **`--num-shards`** — total shards `S`.
* **`--version-id`** — output namespace (e.g. `20260622_r1` = `date_runOfDay`).

## Output layout

```
<output>/<version_id>/
   00000/data-<uuid>.sst
   00001/data-<uuid>.sst
   ...
   _manifest.json    (schema, num_shards, per-shard files + sha256 + row counts)
```

`_manifest.json` is what the **Control Plane** consumes to validate coverage
(every shard present, row counts ≥ thresholds, checksums recorded) before
promoting the version to `activeVersion`.

## Local / Databricks usage

Cluster libs (Databricks: cluster init script or workspace library install):

```
pip install rocksdict protobuf pyspark   # + bharatml_commons (existing internal pkg)
```

Submit:

```
spark-submit --py-files py-sdk/bharatml_commons.zip \
             -m mnemosyne_producer.job \
             --input  gs://.../ds_dbc_ofs_catalog__geo/*.snappy.parquet \
             --config py-sdk/mnemosyne_producer/examples/geo_config.json \
             --output dbfs:/mnt/mnemo/store=catalog__user_geohash_1_3 \
             --num-shards 10 \
             --version-id 20260622_r1
```

`--shuffle-partitions` is exposed as a tunable for very large inputs (it
controls the pre-repartition shuffle width; the final stage always uses
`--num-shards`).

## Smoke test (no Spark)

```
python examples/smoke_test.py
```

Validates:

* config parses
* `RowEncoder` produces decodable `persist.Data` bytes
* defaults applied for null source columns
* `crc32(key) % S` is in range and deterministic

## Module layout

```
mnemosyne_producer/
├── config.py       parse onboarding JSON → Config(entity_label, key_columns, feature_groups[])
├── sharding.py     KEY_SEP="|"  ·  make_key()  ·  shard_id()  ·  crc32_ieee()
├── encoder.py      RowEncoder: row dict → persist.Data → bytes (per-FG typed Values)
├── sst_writer.py   write_partition_ssts(): per-Spark-partition rocksdict.SstFileWriter
└── job.py          PySpark entry point (parse args, build pipeline, write manifest)
```

## V1 scope / not yet

* **Single source table per store.** Multi-source joins (entity features split
  across tables) are deferred — caller must pre-join into one parquet input.
* **Vectors** are encoder-supported but unexercised by the example config.
* **No upload step** — output goes to whatever path you pass (local FS, DBFS
  mount, etc.). For raw `gs://` writes, add an upload hook inside
  `sst_writer._finalize`.
* **Pluggable file format** for the Databricks job is V2 in the LLD; V1 emits
  `persist.Data` proto bytes only.

## Where this fits in Mnemosyne

```
THIS JOB         ────►  GCS (versioned SSTs + manifest)
                                   │
Control Plane    watches GCS  ─────┘
                 mints version, writes shard→pod assignment in etcd
                                   │
                                   ▼
data-loader (engine-agnostic sidecar on each pod)
   pulls assigned SSTs, calls engine.bulk_ingest()
   re-PUTs PodData with WarmVersions += [vId]
                                   │
                                   ▼
Read server (Rust + TCP)  ─►  serves point lookups by key
```

See the [Mnemosyne LLD](https://meesho.atlassian.net/wiki/spaces/EW/pages/5379850241) for the full platform design.
