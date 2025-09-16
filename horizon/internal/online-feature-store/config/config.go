package config

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
)

type Manager interface {
	RegisterStore(confId int, dbType string, table string, primaryKeys []string, tableTtl int) (map[string]interface{}, error)
	CreateStore(storeMap map[string]interface{}) error
	RegisterEntity(entityLabel string, keyMap map[string]Key, distributedCache Cache, inMemoryCache Cache) error
	EditEntity(entityLabel string, distributedCache Cache, inMemoryCache Cache) error
	RegisterFeatureGroup(entityLabel, fgLabel, JobId string, storeId, ttlInSeconds int, inMemoryCacheEnabled, distributedCacheEnabled bool, dataType enums.DataType, featureLabels, featureDefaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string, layoutVersion int) ([]string, Store, map[string]interface{}, error)
	CreateFeatureGroup(featureGroupMap map[string]interface{}) error
	EditFeatureGroup(entityLabel, fgLabel string, ttlInSeconds int, inMemoryCacheEnabled, distributedCacheEnabled bool, layoutVersion int) error
	AddFeatures(entityLabel, featureGroupLabel string, labels, defaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string) ([]string, Store, map[string]interface{}, map[string]interface{}, error)
	CreateAddFeaturesNodes(paths map[string]interface{}, pathsToUpdate map[string]interface{}) error
	EditFeatures(entityLabel, fgLabel string, featureLabels, defaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string) error
	RegisterJob(jobType, jobId, token string) error
	GetEntities() (map[string]Entity, error)
	GetFeatureGroups(entityLabel string) (map[string]FeatureGroup, error)
	GetFeatureGroup(entityLabel, fgLabel string) (*FeatureGroup, error)
	GetAllEntities() ([]string, error)
	GetJobs(jobType string) ([]string, error)
	GetJobsToken(jobType, jobId string) (string, error)
	GetStores() (map[string]Store, error)
	GetAllFeatureGroupByEntityLabel(entityLabel string) ([]string, error)
	GetSource() (map[string]string, error)
	GetEntityKeys(entityLabel string) (map[string]Key, error)
}
