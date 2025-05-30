package compression

import (
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
)

var (
	encoder *ZStdEncoder

	decoder *ZStdDecoder

	mut sync.Mutex
)

type ZStdEncoder struct {
	encoder *zstd.Encoder
}

func NewZStdEncoder() *ZStdEncoder {
	if encoder != nil {
		return encoder
	}
	mut.Lock()
	defer mut.Unlock()
	if encoder != nil {
		return encoder
	}
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		panic(err)
	}
	encoder = &ZStdEncoder{
		encoder: enc,
	}
	return encoder
}

func (e *ZStdEncoder) Encode(data []byte) (cdata []byte) {
	cdata = e.encoder.EncodeAll(data, make([]byte, 0, len(data)))
	log.Debug().Msgf("Original data length: %d", len(data))
	log.Debug().Msgf("Compressed data length: %d", len(cdata))
	return
}

func (e *ZStdEncoder) EncodeV2(data []byte, outputBuffer *[]byte) {
	*outputBuffer = e.encoder.EncodeAll(data, (*outputBuffer)[:0])
	return
}

func (e *ZStdEncoder) EncoderType() Type {
	return TypeZSTD
}

type ZStdDecoder struct {
	decoder *zstd.Decoder
}

func NewZStdDecoder() *ZStdDecoder {
	if decoder != nil {
		return decoder
	}
	mut.Lock()
	defer mut.Unlock()
	if decoder != nil {
		return decoder
	}
	dec, err := zstd.NewReader(nil,
		zstd.WithDecoderConcurrency(0),
		zstd.WithDecoderLowmem(false),
		zstd.IgnoreChecksum(true))
	if err != nil {
		panic(err)
	}
	decoder = &ZStdDecoder{
		decoder: dec,
	}
	return decoder
}

func (d *ZStdDecoder) Decode(cdata []byte) (data []byte, err error) {
	data, err = d.decoder.DecodeAll(cdata, make([]byte, 0, len(cdata)*3))
	if err != nil {
		return
	}
	log.Debug().Msgf("Compressed data length: %d", len(cdata))
	log.Debug().Msgf("Decompressed data length: %d", len(data))
	return

}

func (d *ZStdDecoder) DecoderType() Type {
	return TypeZSTD
}
