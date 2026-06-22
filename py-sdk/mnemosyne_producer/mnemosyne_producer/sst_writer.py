"""
Per-Spark-partition SST writer.

Each Spark partition contains rows already sorted by (shard_id, key).  We
iterate the partition and emit ONE SST file per (partition, shard_id) tuple;
multiple Spark partitions may contribute SSTs to the same shard, all of which
RocksDB's `IngestExternalFile` will absorb on the serving side.

Outputs per partition:
  <out_base>/<version_id>/<shard:05d>/data-<partition_uuid>.sst

A small per-SST manifest dict is yielded back to the driver so we can build the
global `_manifest.json` after `collect()`.

Notes
-----
* Output path is treated as a local-filesystem path; for cloud writes (`gs://`)
  the caller is expected to be running on a worker that has a mount for that
  path (Databricks DBFS) or to add an upload step.  The code is structured so
  that an upload hook can be added easily inside `_finalize`.
* SST file keys MUST be inserted in strictly increasing order — guaranteed by
  the upstream `sortWithinPartitions("__mnemo_shard", "__mnemo_key")`.
"""
from __future__ import annotations

import hashlib
import os
import uuid
from typing import Any, Dict, Iterator

from .encoder import RowEncoder


_SHARD_COL = "__mnemo_shard"
_KEY_COL = "__mnemo_key"


def write_partition_ssts(
    rows_iter: Iterator[Any],
    encoder: RowEncoder,
    out_base: str,
    version_id: str,
) -> Iterator[Dict[str, Any]]:
    """Consume a Spark partition iterator; emit per-shard SST manifests.

    Rows are expected as pyspark `Row` objects with `__mnemo_shard` (int) and
    `__mnemo_key` (str) attributes plus the projected feature columns.
    """
    try:
        from rocksdict import SstFileWriter  # type: ignore
    except ImportError as exc:  # pragma: no cover
        raise RuntimeError(
            "rocksdict is required on Spark workers. Install via cluster "
            "library: `pip install rocksdict`."
        ) from exc

    partition_uuid = uuid.uuid4().hex[:12]

    current_shard: int | None = None
    writer: "SstFileWriter | None" = None
    sst_path: str = ""
    sha: "hashlib._Hash | None" = None
    n_rows: int = 0
    n_bytes: int = 0

    def _finalize() -> Dict[str, Any]:
        """Close the active writer and emit its manifest entry."""
        nonlocal writer, current_shard, sha, n_rows, n_bytes, sst_path
        assert writer is not None
        assert sha is not None
        writer.finish()
        size = os.path.getsize(sst_path) if os.path.exists(sst_path) else None
        manifest = {
            "shard_id": current_shard,
            "path": sst_path,
            "rows": n_rows,
            "bytes_logical": n_bytes,
            "size_on_disk": size,
            "sha256": sha.hexdigest(),
            "partition_uuid": partition_uuid,
        }
        writer = None
        return manifest

    for row in rows_iter:
        # `Row` supports both attribute and dict access.
        row_dict = row.asDict(recursive=False) if hasattr(row, "asDict") else dict(row)
        sid = int(row_dict[_SHARD_COL])
        key_str = row_dict[_KEY_COL]
        key_bytes = key_str.encode("utf-8")
        value_bytes = encoder.encode(row_dict)

        if sid != current_shard:
            if writer is not None:
                yield _finalize()
            current_shard = sid
            shard_dir = os.path.join(out_base, version_id, f"{sid:05d}")
            os.makedirs(shard_dir, exist_ok=True)
            sst_path = os.path.join(shard_dir, f"data-{partition_uuid}.sst")
            writer = SstFileWriter()
            writer.open(sst_path)
            sha = hashlib.sha256()
            n_rows = 0
            n_bytes = 0

        # rocksdict's SstFileWriter supports __setitem__ and put().
        writer[key_bytes] = value_bytes  # type: ignore[index]
        sha.update(key_bytes)             # type: ignore[union-attr]
        sha.update(value_bytes)           # type: ignore[union-attr]
        n_rows += 1
        n_bytes += len(key_bytes) + len(value_bytes)

    if writer is not None:
        yield _finalize()
