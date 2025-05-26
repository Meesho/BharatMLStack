package blocks

import "testing"

func BenchmarkGetPSDBPoolWithoutPool(b *testing.B) {
	_ = GetPSDBPool()
	b.ResetTimer()
	b.ReportAllocs()
	var psdb *PermStorageDataBlock
	for i := 0; i < b.N; i++ {
		psdb = &PermStorageDataBlock{}
		psdb.Clear()
		//psdb.Builder = &PermStorageDataBlockBuilder{psdb: psdb}
	}
	_ = psdb
}

func BenchmarkGetPSDBPoolWithPool(b *testing.B) {
	psdbPool := GetPSDBPool()
	b.ResetTimer()
	b.ReportAllocs()
	var psdb *PermStorageDataBlock
	for i := 0; i < b.N; i++ {
		psdb = psdbPool.Get()
		psdbPool.Put(psdb)
	}
}
