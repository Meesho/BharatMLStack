package compression

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
		f := r.NormFloat64()

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
	enc, err := NewZStdEncoder()
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	var cdata []byte
	enc.Encode(data, &cdata)
	if len(cdata) == 0 || len(cdata) > len(data) {
		t.Errorf("Invalid compressed data length: %d", len(cdata))
	}
}

func TestZStdDecoder_Decode(t *testing.T) {
	data := populateNormFP64Bytes(1000)
	enc, err := NewZStdEncoder()
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	var cdata []byte
	enc.Encode(data, &cdata)
	dec, err := NewZStdDecoder()
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}
	ddata, err := dec.Decode(cdata)
	if err != nil {
		t.Errorf("Error decoding compressed data: %s", err.Error())
	}
	if len(ddata) == 0 || len(data) != len(ddata) {
		t.Errorf("Invalid decoded data length: %d", len(ddata))
	}
}

func TestGetEncoder_ZSTD(t *testing.T) {
	enc, err := GetEncoder(TypeZSTD)
	assert.NoError(t, err)
	assert.NotNil(t, enc)
	var out []byte
	enc.Encode([]byte("test"), &out)
	assert.NotEmpty(t, out)
}

func TestGetEncoder_Unsupported(t *testing.T) {
	enc, err := GetEncoder(TypeNone)
	assert.Error(t, err)
	assert.Nil(t, enc)
	assert.Contains(t, err.Error(), "unsupported compression type")

	enc, err = GetEncoder(Type(99))
	assert.Error(t, err)
	assert.Nil(t, enc)
}

func TestGetDecoder_ZSTD(t *testing.T) {
	dec, err := GetDecoder(TypeZSTD)
	assert.NoError(t, err)
	assert.NotNil(t, dec)
	enc, _ := NewZStdEncoder()
	var compressed []byte
	enc.Encode([]byte("test"), &compressed)
	out, err := dec.Decode(compressed)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), out)
}

func TestGetDecoder_Unsupported(t *testing.T) {
	dec, err := GetDecoder(TypeNone)
	assert.Error(t, err)
	assert.Nil(t, dec)
	assert.Contains(t, err.Error(), "unsupported compression type")
}
