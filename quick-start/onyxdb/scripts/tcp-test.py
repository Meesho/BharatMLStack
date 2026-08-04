#!/usr/bin/env python3
"""
tcp-test.py — Smoke-test the OnyxDB read server TCP binary protocol.

Runs 10 single lookups + 1 batch lookup using real catalog_id/geohash_1_3_id
keys sampled from the loaded parquet dataset.

Usage:
    python3 scripts/tcp-test.py [--host 127.0.0.1] [--port 9091]
"""
import argparse, socket, struct, sys

try:
    import zstandard, msgpack
    DECODE = True
except ImportError:
    DECODE = False
    print("[warn] zstandard/msgpack not installed — values will be printed as raw bytes")

# ── Protocol helpers ─────────────────────────────────────────────────────────

def make_key(catalog_id: int, geohash_1_3_id: int) -> bytes:
    """Canonical fixed 12-byte OnyxDB key: key_a = uint64 BE, key_b = uint32 BE.

    This is the SINGLE key construction shared across the whole pipeline and MUST be
    byte-identical everywhere or every lookup misses:
      * read server  — readserver/src/tcp/protocol.rs: [8B key_a BE][4B key_b BE], KEY_SIZE=12
                        (its make_key copies key_a.to_be_bytes() then key_b.to_be_bytes())
      * Go SDK       — onyxdb-go-sdk builds this same []byte and shards by crc32 over it
      * Spark producer — must write RocksDB keyed by these exact 12 bytes (not a string key)
    key_a = catalog_id (u64), key_b = geohash_1_3_id (u32), both big-endian, no entity-label
    prefix and no hex — the server uses the raw key bytes directly as the RocksDB key.
    """
    return struct.pack(">QI", catalog_id & 0xFFFFFFFFFFFFFFFF, geohash_1_3_id & 0xFFFFFFFF)

def recv_exact(sock, n: int) -> bytes:
    buf = b""
    while len(buf) < n:
        chunk = sock.recv(n - len(buf))
        if not chunk:
            raise ConnectionError("socket closed before receiving all bytes")
        buf += chunk
    return buf

def single_lookup(sock, key: bytes) -> bytes | None:
    sock.sendall(b"\x01" + key)
    found = recv_exact(sock, 1)[0]
    if not found:
        return None
    length = struct.unpack(">I", recv_exact(sock, 4))[0]
    return recv_exact(sock, length)

def batch_lookup(sock, keys: list[bytes]) -> list[bytes | None]:
    n = len(keys)
    payload = b"\x02" + struct.pack(">H", n) + b"".join(keys)
    sock.sendall(payload)
    resp_n = struct.unpack(">H", recv_exact(sock, 2))[0]
    results = []
    
    for _ in range(resp_n):
        found = recv_exact(sock, 1)[0]
        if found:
            length = struct.unpack(">I", recv_exact(sock, 4))[0]
            results.append(recv_exact(sock, length))
        else:
            results.append(None)
    return results

def decode_value(raw: bytes) -> str:
    if not DECODE:
        return f"<{len(raw)} bytes>"
    try:
        dec = zstandard.decompress(raw)
        data = msgpack.unpackb(dec, raw=False)
        g1 = data.get("g1", "")
        g3 = data.get("g3", "")
        return f"g1={g1!r} g3={g3!r} features={len(data.get('f', []))}"
    except Exception as e:
        return f"<{len(raw)} bytes, decode error: {e}>"

# ── Test keys (real samples from the loaded parquet) ─────────────────────────

TEST_KEYS = [
    (105959719, 4236),
    (10000001, 4208)
]

# ── Main ─────────────────────────────────────────────────────────────────────

def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=9091)
    args = ap.parse_args()

    print(f"Connecting to {args.host}:{args.port} ...")
    with socket.create_connection((args.host, args.port), timeout=5) as s:
        s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        print(f"Connected.\n")

        # ── Single lookups ────────────────────────────────────────────────────
        print("─" * 60)
        print(f"{'Single lookups':^60}")
        print("─" * 60)

        hits = 0
        for cid, ghid in TEST_KEYS:
            key = make_key(cid, ghid)
            val = single_lookup(s, key)
            status = "HIT " if val else "miss"
            if val:
                hits += 1
                detail = decode_value(val)
                print(f"  [{status}] cat={cid:>12}  geo={ghid:>5}  {detail}")
            else:
                print(f"  [{status}] cat={cid:>12}  geo={ghid:>5}")

        print(f"\n  Result: {hits}/{len(TEST_KEYS)} hits")

        # ── Batch lookup ──────────────────────────────────────────────────────
        print(f"\n{'─' * 60}")
        print(f"{'Batch lookup (' + str(len(TEST_KEYS)) + ' keys)':^60}")
        print("─" * 60)

        keys = [make_key(cid, ghid) for cid, ghid in TEST_KEYS]
        results = batch_lookup(s, keys)
        batch_hits = sum(1 for r in results if r is not None)
        print(f"  Batch result: {batch_hits}/{len(results)} hits")

        # ── Summary ───────────────────────────────────────────────────────────
        print(f"\n{'─' * 60}")
        if hits > 0 and hits == batch_hits:
            print("✓  TCP connectivity OK — single and batch results match")
        elif hits > 0:
            print("✓  TCP connectivity OK (some hits found)")
        else:
            print("!  No hits found — check that data is loaded and activated")
            print("   Run: docker compose up  (init container will load/activate)")
        print("─" * 60)

if __name__ == "__main__":
    main()
