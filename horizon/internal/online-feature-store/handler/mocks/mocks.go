package mocks

import (
	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/features"
	"github.com/stretchr/testify/mock"
)

type ConfigManager struct {
	mock.Mock
}

func (m *ConfigManager) GetFeatureGroup(entityLabel, featureGroupLabel string) (*ofsConfig.FeatureGroup, error) {
	args := m.Called(entityLabel, featureGroupLabel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ofsConfig.FeatureGroup), args.Error(1)
}

func (m *ConfigManager) AddFeatures(a, b string, c, d, e, f, g, h, i []string) ([]string, ofsConfig.Store, map[string]interface{}, map[string]interface{}, error) {
	return nil, ofsConfig.Store{}, nil, nil, nil
}

func (m *ConfigManager) CreateStore(storeMap map[string]interface{}) error { return nil }
func (m *ConfigManager) CreateAddFeaturesNodes(paths map[string]interface{}, pathsToUpdate map[string]interface{}) error {
	return nil
}
func (m *ConfigManager) CreateFeatureGroup(featureGroupMap map[string]interface{}) error {
	return nil
}
func (m *ConfigManager) DeleteFeatures(entityLabel, featureGroupLabel string, featureLabels []string) error {
	args := m.Called(entityLabel, featureGroupLabel, featureLabels)
	return args.Error(0)
}
func (m *ConfigManager) EditEntity(entityLabel string, distributedCache ofsConfig.Cache, inMemoryCache ofsConfig.Cache) error {
	return nil
}
func (m *ConfigManager) EditFeatureGroup(entityLabel, fgLabel string, ttlInSeconds int, inMemoryCacheEnabled, distributedCacheEnabled bool, layoutVersion int) error {
	return nil
}
func (m *ConfigManager) EditFeatures(entityLabel, fgLabel string, featureLabels, defaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string) error {
	return nil
}
func (m *ConfigManager) GetAllEntities() ([]string, error) { return nil, nil }
func (m *ConfigManager) GetAllFeatureGroupByEntityLabel(entityLabel string) ([]string, error) {
	return nil, nil
}
func (m *ConfigManager) GetEntities() (map[string]ofsConfig.Entity, error) { return nil, nil }
func (m *ConfigManager) GetEntityKeys(entityLabel string) (map[string]ofsConfig.Key, error) {
	return nil, nil
}
func (m *ConfigManager) GetFeatureGroups(entityLabel string) (map[string]ofsConfig.FeatureGroup, error) {
	return nil, nil
}
func (m *ConfigManager) GetJobs(jobType string) ([]string, error) { return nil, nil }
func (m *ConfigManager) GetJobsToken(jobType string, jobId string) (string, error) {
	return "", nil
}
func (m *ConfigManager) GetSource() (map[string]string, error)          { return nil, nil }
func (m *ConfigManager) GetStores() (map[string]ofsConfig.Store, error) { return nil, nil }
func (m *ConfigManager) RegisterEntity(entityLabel string, keyMap map[string]ofsConfig.Key, distributedCache ofsConfig.Cache, inMemoryCache ofsConfig.Cache) error {
	return nil
}
func (m *ConfigManager) RegisterFeatureGroup(entityLabel, fgLabel, JobId string, storeId, ttlInSeconds int, inMemoryCacheEnabled, distributedCacheEnabled bool, dataType enums.DataType, featureLabels, featureDefaultValues, storageProvider, sourceBasePath, sourceDataPath, stringLength, vectorLength []string, layoutVersion int) ([]string, ofsConfig.Store, map[string]interface{}, error) {
	return nil, ofsConfig.Store{}, nil, nil
}
func (m *ConfigManager) RegisterJob(jobType, jobId, token string) error { return nil }
func (m *ConfigManager) RegisterStore(confId int, dbType string, table string, primaryKeys []string, tableTtl int) (map[string]interface{}, error) {
	return nil, nil
}

type FeatureRepository struct {
	mock.Mock
}

func (m *FeatureRepository) Create(feature *features.Table) (uint, error) {
	args := m.Called(feature)
	return uint(args.Int(0)), args.Error(1)
}

func (m *FeatureRepository) GetAll() ([]features.Table, error) { return nil, nil }
func (m *FeatureRepository) GetAllByUserId(userId string) ([]features.Table, error) {
	return nil, nil
}
func (m *FeatureRepository) GetById(id int) (*features.Table, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*features.Table), args.Error(1)
}
func (m *FeatureRepository) Update(feature *features.Table) error {
	args := m.Called(feature)
	return args.Error(0)
}
