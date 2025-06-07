package compression

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	m.Run()
}

func populateNormFP64Bytes(num int) []byte {
	// Create a byte slice large enough to hold `num` float32 values (4 bytes each)
	bytes := make([]byte, num*8)

	// Seed the random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < num; i++ {
		// Generate a random float32 number
		f := rand.NormFloat64()

		// Get the binary representation of the float32 number as a uint32
		bits := math.Float64bits(f)

		// Use bit shifting to split the uint32 into 4 bytes
		bytes[i*4] = byte(bits >> 56)   // Most significant byte
		bytes[i*4+1] = byte(bits >> 48) // Next byte
		bytes[i*4+2] = byte(bits >> 40) // Next byte
		bytes[i*4+3] = byte(bits >> 32) // Least significant byte
		bytes[i*4+4] = byte(bits >> 24) // Least significant byte
		bytes[i*4+5] = byte(bits >> 16) // Least significant byte
		bytes[i*4+6] = byte(bits >> 8)  // Least significant byte
		bytes[i*4+7] = byte(bits)       // Least significant byte
	}

	return bytes
}
func TestZStdEncoder_Encode(t *testing.T) {
	data := populateNormFP64Bytes(1000)
	enc := NewZStdEncoder()
	cdata := enc.Encode(data)
	if len(cdata) == 0 || len(cdata) > len(data) {
		t.Errorf("Invalid compressed data length: %d", len(cdata))
	}
}

func TestZStdDecoder_Decode(t *testing.T) {
	data := populateNormFP64Bytes(1000)
	enc := NewZStdEncoder()
	cdata := enc.Encode(data)
	dec := NewZStdDecoder()
	ddata, err := dec.Decode(cdata)
	if err != nil {
		t.Errorf("Error decoding compressed data: %s", err.Error())
	}
	if len(ddata) == 0 || len(data) != len(ddata) {
		t.Errorf("Invalid decoded data length: %d", len(ddata))
	}
}
