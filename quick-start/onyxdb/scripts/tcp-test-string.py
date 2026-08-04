#!/usr/bin/env python3
"""
tcp-test-string.py — Smoke-test the OnyxDB read server with OFS string keys.

The canonical OnyxDB key is an OFS-compatible variable-length STRING key:
    key = "<entity_label>:<catalog_id>|<geohash_1_3_id>"   (UTF-8 bytes)
    e.g. "catalog__user_geohash_1_3:105959719|4236"

This is the same key OFS writes to Redis (buildCacheKeyForPersist). The read
server supports string keys via dedicated opcodes:
    Single (opcode 0x03):  [1B op=0x03][2B keyLen BE][key bytes]
    Batch  (opcode 0x04):  [1B op=0x04][2B N BE][ per key: 2B keyLen BE + key bytes ]
    Response framing is unchanged: [1B found][4B len BE][value bytes].

Usage:
    python3 scripts/tcp-test-string.py [--host 127.0.0.1] [--port 9091] \
        [--entity-label catalog__user_geohash_1_3]
"""
import argparse, socket, struct, sys

try:
    import zstandard, msgpack
    DECODE = True
except ImportError:
    DECODE = False
    print("[warn] zstandard/msgpack not installed — values will be printed as raw bytes")

# ── Key construction (OFS string key) ────────────────────────────────────────

DEFAULT_ENTITY_LABEL = "catalog__user_geohash_1_3"  # OFS entity label (double underscores)

def make_string_key(catalog_id: int, geohash_1_3_id: int, entity_label: str) -> bytes:
    """OFS-compatible string key, UTF-8: '<entity_label>:<catalog_id>|<geohash_1_3_id>'.

    Mirrors OFS buildCacheKeyForPersist and the producer's build_string_key — pk values are
    plain decimal integers joined by '|', with the entity-label prefix. No hex, no padding.
    """
    return f"{entity_label}:{catalog_id}|{geohash_1_3_id}".encode("utf-8")

# ── Protocol helpers ──────────────────────────────────────────────────────────

OP_SINGLE = 0x03
OP_BATCH  = 0x04

def recv_exact(sock, n: int) -> bytes:
    buf = b""
    while len(buf) < n:
        chunk = sock.recv(n - len(buf))
        if not chunk:
            raise ConnectionError("socket closed before receiving all bytes")
        buf += chunk
    return buf

def single_lookup(sock, key: bytes) -> bytes | None:
    # [1B op][2B keyLen BE][key]
    sock.sendall(bytes([OP_SINGLE]) + struct.pack(">H", len(key)) + key)
    found = recv_exact(sock, 1)[0]
    if not found:
        return None
    length = struct.unpack(">I", recv_exact(sock, 4))[0]
    return recv_exact(sock, length)

def batch_lookup(sock, keys: list[bytes]) -> list[bytes | None]:
    # [1B op][2B N BE][ per key: 2B keyLen BE + key ]
    payload = bytes([OP_BATCH]) + struct.pack(">H", len(keys))
    for k in keys:
        payload += struct.pack(">H", len(k)) + k
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
    (100000566, 4250),              # from SST dump — first key
    (10000001, 4208),       # from SST dump — second key
    (100000218, 5006),      # from SST dump — third key
    (100001688, 4608),      # from SST dump — fourth key
    (999999999, 1),         # likely miss — verifies protocol handles misses
]

# ── Main ─────────────────────────────────────────────────────────────────────

def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=9091)
    ap.add_argument("--entity-label", default=DEFAULT_ENTITY_LABEL,
                    help="OFS entity label prefix for the string key")
    args = ap.parse_args()

    print(f"Connecting to {args.host}:{args.port} (string-key, length-prefixed) ...")
    with socket.create_connection((args.host, args.port), timeout=5) as s:
        s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        print("Connected.\n")

        # ── Single lookups ────────────────────────────────────────────────────
        print("-" * 60)
        print(f"{'Single lookups (string key)':^60}")
        print("-" * 60)

        hits = 0
        for cid, ghid in TEST_KEYS:
            key = make_string_key(cid, ghid, args.entity_label)
            val = single_lookup(s, key)
            status = "HIT " if val else "miss"
            if val:
                hits += 1
                print(f"  [{status}] {key.decode()}  {decode_value(val)}")
            else:
                print(f"  [{status}] {key.decode()}")

        print(f"\n  Result: {hits}/{len(TEST_KEYS)} hits")

        # ── Batch lookup ──────────────────────────────────────────────────────
        print(f"\n{'-' * 60}")
        print(f"{'Batch lookup (' + str(len(TEST_KEYS)) + ' string keys)':^60}")
        print("-" * 60)

        keys = [make_string_key(cid, ghid, args.entity_label) for cid, ghid in TEST_KEYS]
        results = batch_lookup(s, keys)
        batch_hits = sum(1 for r in results if r is not None)
        print(f"  Batch result: {batch_hits}/{len(results)} hits")

        # ── Summary ───────────────────────────────────────────────────────────
        print(f"\n{'-' * 60}")
        if hits > 0 and hits == batch_hits:
            print("  TCP connectivity OK — single and batch results match")
        elif hits > 0:
            print("  TCP connectivity OK (some hits found)")
        else:
            print("!  No hits found — check the data was written with string keys,")
            print("   and the version is loaded + activated.")
        print("-" * 60)

if __name__ == "__main__":
    main()
