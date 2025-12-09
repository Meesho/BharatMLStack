package config

import "github.com/Meesho/BharatMLStack/online-feature-store/pkg/circuitbreaker"

type Manager interface {
	GetEntity(entityLabel string) (*Entity, error)
	GetAllEntities() []Entity
	GetFeatureGroup(entityLabel, fgLabel string) (*FeatureGroup, error)
	GetStore(storeId string) (*Store, error)
	GetFeatureGroupColumn(entityLabel, fgLabel, columnLabel string) (*Column, error)
	GetDistributedCacheConfForEntity(entityLabel string) (*Cache, error)
	GetInMemoryCacheConfForEntity(entityLabel string) (*Cache, error)
	GetP2PCacheConfForEntity(entityLabel string) (*Cache, error)
	GetP2PEnabledPercentage() int
	GetStores() (*map[string]Store, error)
	GetActiveFeatureSchema(entityLabel, fgLabel string) (*FeatureSchema, error)
	GetColumnsForEntityAndFG(entityLabel string, fgId int) ([]string, error)
	GetMaxColumnSize(storeId string) (int, error)
	GetPKMapAndPKColumnsForEntity(entityLabel string) (map[string]string, []string, error)
	GetDefaultValueByte(entityLabel string, fgId int, version int, featureLabel string) ([]byte, error)
	GetSequenceNo(entityLabel string, fgId int, version int, featureLabel string) (int, error)
	GetStringLengths(entityLabel string, fgId int, version int) ([]uint16, error)
	GetVectorLengths(entityLabel string, fgId int, version int) ([]uint16, error)
	GetNumOfFeatures(entityLabel string, fgId int, version int) (int, error)
	GetActiveVersion(entityLabel string, fgId int) (int, error)
	GetNormalizedEntities() error
	RegisterClients() error
	GetAllRegisteredClients() map[string]string
	GetAllFGIdsForEntity(entityLabel string) (map[int]bool, error)
	GetCircuitBreakerConfigs() map[string]circuitbreaker.Config
	UpdateCBConfigs() error
}
