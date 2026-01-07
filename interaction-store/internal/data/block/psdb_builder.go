package blocks

import (
	"github.com/Meesho/BharatMLStack/interaction-store/internal/compression"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/enum"
)

const (
	PSDBLayout1HeaderLength = 2
)

type PermanentStorageDataBlockBuilder struct {
	psdb *PermanentStorageDataBlock
}

func NewPermanentStorageDataBlockBuilder() *PermanentStorageDataBlockBuilder {
	return &PermanentStorageDataBlockBuilder{
		psdb: &PermanentStorageDataBlock{},
	}
}

func (p *PermanentStorageDataBlockBuilder) SetLayoutVersion(version uint8) *PermanentStorageDataBlockBuilder {
	p.psdb.LayoutVersion = version
	return p
}

func (p *PermanentStorageDataBlockBuilder) SetCompressionType(compressionType compression.Type) *PermanentStorageDataBlockBuilder {
	p.psdb.CompressionType = compressionType
	return p
}

func (p *PermanentStorageDataBlockBuilder) SetData(data any) *PermanentStorageDataBlockBuilder {
	p.psdb.Data = data
	return p
}

func (p *PermanentStorageDataBlockBuilder) SetDataLength(dataLength uint16) *PermanentStorageDataBlockBuilder {
	p.psdb.DataLength = dataLength
	return p
}

func (p *PermanentStorageDataBlockBuilder) SetInteractionType(interactionType enum.InteractionType) *PermanentStorageDataBlockBuilder {
	p.psdb.InteractionType = interactionType
	return p
}

func (p *PermanentStorageDataBlockBuilder) Build() *PermanentStorageDataBlock {
	p.psdb.Buf = make([]byte, PSDBLayout1HeaderLength)
	p.psdb.OriginalData = make([]byte, 0)
	return p.psdb
}
