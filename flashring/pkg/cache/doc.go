// Package cache is the public entry point of flashring. It exposes
// three implementations of the Cache interface:
//
//   - WrapCache: the production sharded NVMe-backed cache. Index in
//     RAM, data in O_DIRECT append-only files, I/O batched through
//     io_uring. Linux only. Constructed via NewWrapCache, which wipes
//     the supplied mount point.
//   - Badger: a thin adapter over dgraph-io/badger used as a
//     comparison backend in the benchmark harness. Not for production.
//   - Freecache: a thin adapter over coocood/freecache used as a
//     comparison backend. NewFreecache lowers GOGC process-wide.
//
// MGet is unique to WrapCache and not on the interface — it coalesces
// nearby disk reads into a single io_uring submission, so batching
// keys through MGet is materially faster than concurrent Get calls.
package cache
