package byteorder

import "testing"

func TestUint32Uint64PackUnpack(t *testing.T) {
	tests := []struct {
		high uint32
		low  uint32
	}{
		{0, 0},
		{1, 2},
		{0xFFFFFFFF, 0xFFFFFFFF},
		{0x12345678, 0x9ABCDEF0},
	}

	for _, tt := range tests {
		packed := BinPackUint32InUint64(tt.high, tt.low)
		h, l := UnpackUint64InUint32(packed)
		if h != tt.high || l != tt.low {
			t.Errorf("Pack/Unpack mismatch: high=%08X low=%08X => packed=%016X => got_high=%08X got_low=%08X", tt.high, tt.low, packed, h, l)
		}
	}
}

func TestUint16Uint32PackUnpack(t *testing.T) {
	tests := []struct {
		high uint16
		low  uint16
	}{
		{0, 0},
		{1, 2},
		{0xFFFF, 0xFFFF},
		{0xABCD, 0x1234},
	}

	for _, tt := range tests {
		packed := BinPackUint16InUint32(tt.high, tt.low)
		h, l := UnpackUint32InUint16(packed)
		if h != tt.high || l != tt.low {
			t.Errorf("Pack/Unpack mismatch: high=%04X low=%04X => packed=%08X => got_high=%04X got_low=%04X", tt.high, tt.low, packed, h, l)
		}
	}
}

func TestUint8Uint16PackUnpack(t *testing.T) {
	tests := []struct {
		high uint8
		low  uint8
	}{
		{0, 0},
		{1, 2},
		{0xFF, 0xFF},
		{0xAB, 0xCD},
	}

	for _, tt := range tests {
		packed := BinPackUint8InUint16(tt.high, tt.low)
		h, l := UnpackUint16InUint8(packed)
		if h != tt.high || l != tt.low {
			t.Errorf("Pack/Unpack mismatch: high=%02X low=%02X => packed=%04X => got_high=%02X got_low=%02X", tt.high, tt.low, packed, h, l)
		}
	}
}
