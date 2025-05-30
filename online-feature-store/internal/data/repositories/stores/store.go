package stores

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/models"
)

const (
	StoreTypeRedis  = "redis"
	StoreTypeScylla = "scylla"
)

type Store interface {
	PersistV2(storeId string, entityLabel string, pkMap map[string]string, fgIdToPsDb map[int]*blocks.PermStorageDataBlock) error
	RetrieveV2(entityLabel string, pkMap map[string]string, fgIds []int) (map[int]*blocks.DeserializedPSDB, error)
	BatchPersistV2(storeId string, entityLabel string, rows []models.Row) error
	BatchRetrieveV2(entityLabel string, pkMaps []map[string]string, fgIds []int) ([]map[int]*blocks.DeserializedPSDB, error)
	Type() string
}
