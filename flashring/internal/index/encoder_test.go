package index

import "testing"

func TestEncodeDecode_RoundTrips_AllFields(t *testing.T) {
	var entry Entry

	encode("ignored", 0x1234, 0xABCD, 0x5678, 0x9876, 0xDEADBEEF, 0xCAFEBABE, &entry)

	length, deltaExptime, lastAccess, freq, memId, offset := decode(&entry)
	if length != 0x1234 {
		t.Fatalf("length = %x, want 0x1234", length)
	}
	if deltaExptime != 0xABCD {
		t.Fatalf("deltaExptime = %x, want 0xABCD", deltaExptime)
	}
	if lastAccess != 0x5678 {
		t.Fatalf("lastAccess = %x, want 0x5678", lastAccess)
	}
	if freq != 0x9876 {
		t.Fatalf("freq = %x, want 0x9876", freq)
	}
	if memId != 0xDEADBEEF {
		t.Fatalf("memId = %x, want 0xDEADBEEF", memId)
	}
	if offset != 0xCAFEBABE {
		t.Fatalf("offset = %x, want 0xCAFEBABE", offset)
	}
}

func TestEncodeDecode_ZeroValues(t *testing.T) {
	var entry Entry

	encode("k", 0, 0, 0, 0, 0, 0, &entry)

	length, deltaExptime, lastAccess, freq, memId, offset := decode(&entry)
	if length != 0 || deltaExptime != 0 || lastAccess != 0 || freq != 0 || memId != 0 || offset != 0 {
		t.Fatalf("zero round-trip = (%d,%d,%d,%d,%d,%d), want all zero",
			length, deltaExptime, lastAccess, freq, memId, offset)
	}
}

func TestDecodeMemIdOffset_MatchesDecode(t *testing.T) {
	var entry Entry
	encode("k", 1, 2, 3, 4, 100, 200, &entry)

	memId, offset := DecodeMemIdOffset(&entry)
	if memId != 100 {
		t.Fatalf("memId = %d, want 100", memId)
	}
	if offset != 200 {
		t.Fatalf("offset = %d, want 200", offset)
	}
}

func TestEncodeLastAccessNFreq_PreservesOtherFields(t *testing.T) {
	var entry Entry
	encode("k", 1234, 5678, 100, 50, 0x11111111, 0x22222222, &entry)

	encodeLastAccessNFreq(0xABCD, 0x1234, &entry)

	length, deltaExptime, lastAccess, freq, memId, offset := decode(&entry)
	if length != 1234 {
		t.Fatalf("length corrupted: %d, want 1234", length)
	}
	if deltaExptime != 5678 {
		t.Fatalf("deltaExptime corrupted: %d, want 5678", deltaExptime)
	}
	if lastAccess != 0xABCD {
		t.Fatalf("lastAccess = %x, want 0xABCD", lastAccess)
	}
	if freq != 0x1234 {
		t.Fatalf("freq = %x, want 0x1234", freq)
	}
	if memId != 0x11111111 {
		t.Fatalf("memId corrupted: %x, want 0x11111111", memId)
	}
	if offset != 0x22222222 {
		t.Fatalf("offset corrupted: %x, want 0x22222222", offset)
	}
}

func TestEncodeHashNextPrev_SetsAllFields(t *testing.T) {
	var entry HashNextPrev
	encodeHashNextPrev(0x1111, 0x2222, 5, 7, &entry)

	if entry[0] != 0x1111 {
		t.Fatalf("hhi = %x, want 0x1111", entry[0])
	}
	if entry[1] != 0x2222 {
		t.Fatalf("hlo = %x, want 0x2222", entry[1])
	}
	if decodePrev(&entry) != 5 {
		t.Fatalf("prev = %d, want 5", decodePrev(&entry))
	}
	if decodeNext(&entry) != 7 {
		t.Fatalf("next = %d, want 7", decodeNext(&entry))
	}
	if decodeHashLo(&entry) != 0x2222 {
		t.Fatalf("hashLo = %x, want 0x2222", decodeHashLo(&entry))
	}
}

func TestEncodeUpdatePrev_KeepsNext(t *testing.T) {
	var entry HashNextPrev
	encodeHashNextPrev(0x1, 0x2, 10, 20, &entry)

	encodeUpdatePrev(99, &entry)

	if decodePrev(&entry) != 99 {
		t.Fatalf("prev = %d, want 99", decodePrev(&entry))
	}
	if decodeNext(&entry) != 20 {
		t.Fatalf("next changed: %d, want 20", decodeNext(&entry))
	}
}

func TestEncodeUpdateNext_KeepsPrev(t *testing.T) {
	var entry HashNextPrev
	encodeHashNextPrev(0x1, 0x2, 10, 20, &entry)

	encodeUpdateNext(77, &entry)

	if decodeNext(&entry) != 77 {
		t.Fatalf("next = %d, want 77", decodeNext(&entry))
	}
	if decodePrev(&entry) != 10 {
		t.Fatalf("prev changed: %d, want 10", decodePrev(&entry))
	}
}

func TestEncodeUpdatePrev_NegativeOne(t *testing.T) {
	var entry HashNextPrev
	encodeHashNextPrev(0x1, 0x2, 10, 20, &entry)

	encodeUpdatePrev(-1, &entry)

	if decodePrev(&entry) != -1 {
		t.Fatalf("prev = %d, want -1", decodePrev(&entry))
	}
	if decodeNext(&entry) != 20 {
		t.Fatalf("next changed: %d, want 20", decodeNext(&entry))
	}
}

func TestEncodeUpdateNext_NegativeOne(t *testing.T) {
	var entry HashNextPrev
	encodeHashNextPrev(0x1, 0x2, 10, 20, &entry)

	encodeUpdateNext(-1, &entry)

	if decodeNext(&entry) != -1 {
		t.Fatalf("next = %d, want -1", decodeNext(&entry))
	}
	if decodePrev(&entry) != 10 {
		t.Fatalf("prev changed: %d, want 10", decodePrev(&entry))
	}
}
