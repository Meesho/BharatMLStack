package compression

import (
	"sync"

	"github.com/klauspost/compress/zstd"
)

var (
	encoder *ZStdEncoder

	decoder *ZStdDecoder

	mut sync.Mutex
)

type ZStdEncoder struct {
	encoder *zstd.Encoder
}

func NewZStdEncoder() (*ZStdEncoder, error) {
	if encoder != nil {
		return encoder, nil
	}
	mut.Lock()
	defer mut.Unlock()
	if encoder != nil {
		return encoder, nil
	}
	// SpeedBetterCompression will yield better compression than the default.
	// Currently it is about zstd level 7-8 with ~ 2x-3x the default CPU usage.
	// By using this, notice that CPU usage may go up in the future.
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		return nil, err
	}
	encoder = &ZStdEncoder{
		encoder: enc,
	}
	return encoder, nil
}

func (e *ZStdEncoder) Encode(data []byte, outputBuffer *[]byte) {
	*outputBuffer = e.encoder.EncodeAll(data, (*outputBuffer)[:0])
}

type ZStdDecoder struct {
	decoder *zstd.Decoder
}

func NewZStdDecoder() (*ZStdDecoder, error) {
	if decoder != nil {
		return decoder, nil
	}
	mut.Lock()
	defer mut.Unlock()
	if decoder != nil {
		return decoder, nil
	}
	//When a value of 0 is provided in DecoderConcurrency, GOMAXPROCS will be used
	dec, err := zstd.NewReader(nil,
		zstd.WithDecoderConcurrency(0),
		zstd.WithDecoderLowmem(false),
		zstd.IgnoreChecksum(true))
	if err != nil {
		return nil, err
	}
	decoder = &ZStdDecoder{
		decoder: dec,
	}
	return decoder, nil
}

func (d *ZStdDecoder) Decode(cdata []byte) (data []byte, err error) {
	data, err = d.decoder.DecodeAll(cdata, make([]byte, 0, len(cdata)*3))
	if err != nil {
		return
	}
	return
}
