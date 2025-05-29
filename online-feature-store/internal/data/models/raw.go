package models

import "github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"

type Row struct {
	PkMap      map[string]string
	FgIdToPsDb map[int]*blocks.PermStorageDataBlock
}
