package metrics

import "strconv"

const (
	KEY_GET_LATENCY     = "flashring_get_latency"
	KEY_PUT_LATENCY     = "flashring_put_latency"
	KEY_RTHROUGHPUT     = "flashring_rthroughput"
	KEY_WTHROUGHPUT     = "flashring_wthroughput"
	KEY_HITRATE         = "flashring_hitrate"
	KEY_ACTIVE_ENTRIES  = "flashring_active_entries"
	KEY_EXPIRED_ENTRIES = "flashring_expired_entries"
	KEY_REWRITES        = "flashring_rewrites"
	KEY_GETS            = "flashring_gets"
	KEY_PUTS            = "flashring_puts"
	KEY_HITS            = "flashring_hits"

	KEY_KEY_NOT_FOUND_COUNT = "flashring_key_not_found_count"
	KEY_KEY_EXPIRED_COUNT   = "flashring_key_expired_count"
	KEY_BAD_DATA_COUNT      = "flashring_bad_data_count"
	KEY_BAD_LENGTH_COUNT    = "flashring_bad_length_count"
	KEY_BAD_CR32_COUNT      = "flashring_bad_cr32_count"
	KEY_BAD_KEY_COUNT       = "flashring_bad_key_count"
	KEY_DELETED_KEY_COUNT   = "flashring_deleted_key_count"

	TAG_LATENCY_PERCENTILE = "latency_percentile"
	TAG_VALUE_P25          = "p25"
	TAG_VALUE_P50          = "p50"
	TAG_VALUE_P99          = "p99"
	TAG_SHARD_IDX          = "shard_idx"
	TAG_MEMTABLE_ID        = "memtable_id"

	KEY_WRITE_COUNT      = "flashring_write_count"
	KEY_PUNCH_HOLE_COUNT = "flashring_punch_hole_count"
	KEY_PREAD_COUNT      = "flashring_pread_count"

	KEY_TRIM_HEAD_LATENCY = "flashring_wrap_file_trim_head_latency"
	KEY_PREAD_LATENCY     = "flashring_pread_latency"
	KEY_PWRITE_LATENCY    = "flashring_pwrite_latency"

	KEY_MEMTABLE_FLUSH_COUNT = "flashring_memtable_flush_count"

	LATENCY_RLOCK = "flashring_rlock_latency"
	LATENCY_WLOCK = "flashring_wlock_latency"

	KEY_RINGBUFFER_ACTIVE_ENTRIES = "flashring_ringbuffer_active_entries"

	KEY_MEMTABLE_ENTRY_COUNT = "flashring_memtable_entry_count"

	KEY_MEMTABLE_HIT  = "flashring_memtable_hit"
	KEY_MEMTABLE_MISS = "flashring_memtable_miss"

	KEY_DATA_LENGTH = "flashring_data_length"

	KEY_IOURING_SIZE = "flashring_iouring_size"
)

func GetShardTag(shardIdx uint32) []string {
	return BuildTag(NewTag(TAG_SHARD_IDX, strconv.Itoa(int(shardIdx))))
}

func GetMemtableTag(memtableId uint32) []string {
	return BuildTag(NewTag(TAG_MEMTABLE_ID, strconv.Itoa(int(memtableId))))
}
