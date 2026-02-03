package predatorconfig

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type PredatorConfigRepository interface {
	Create(config *PredatorConfig) error
	Update(config *PredatorConfig) error
	GetByID(id int) (*PredatorConfig, error)
	GetActiveModelByIds(ids []int) ([]PredatorConfig, error)
	GetByDiscoveryConfigID(DiscoveryConfigID int) ([]PredatorConfig, error)
	FindAllActiveConfig() ([]PredatorConfig, error)
	FindAllActiveConfigByEmail(email string) ([]PredatorConfig, error)
	BulkDeactivateByModelNames(modelNames []string, updatedBy string, discoveryConfigId []int) error
	FindByDiscoveryIDsAndCreatedBefore(discoveryIDs []int, daysAgo int) ([]PredatorConfig, error)
	DB() *gorm.DB
	WithTx(tx *gorm.DB) PredatorConfigRepository
	GetByModelName(modelName string) (*PredatorConfig, error)
	GetActiveModelByModelName(modelName string) (*PredatorConfig, error)
	GetActiveModelByModelNameList(modelNames []string) ([]PredatorConfig, error)
	FindByDiscoveryIDsAndAge(discoveryConfigIds []int, daysAgo int) ([]PredatorConfig, error)
}

type predatorConfigRepo struct {
	db *gorm.DB
}

func (r *predatorConfigRepo) GetActiveModelByModelNameList(modelNames []string) ([]PredatorConfig, error) {
	var configs []PredatorConfig
	err := r.db.Where("model_name IN ? AND active = ?", modelNames, true).Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *predatorConfigRepo) GetActiveModelByModelName(modelName string) (*PredatorConfig, error) {
	var config PredatorConfig
	err := r.db.Where("model_name = ? AND active = ?", modelName, true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func NewRepository(connection *infra.SQLConnection) (PredatorConfigRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &predatorConfigRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *predatorConfigRepo) Create(config *PredatorConfig) error {
	return r.db.Create(config).Error
}

func (r *predatorConfigRepo) Update(config *PredatorConfig) error {
	return r.db.Model(config).
		Where("id = ?", config.ID).
		UpdateColumns(map[string]interface{}{
			"active":       config.Active,
			"meta_data":    config.MetaData,
			"updated_by":   config.UpdatedBy,
			"updated_at":   config.UpdatedAt,
			"test_results": config.TestResults,
			"has_nil_data": config.HasNilData,
			"source_model_name": config.SourceModelName,
		}).Error
}

func (r *predatorConfigRepo) GetByID(id int) (*PredatorConfig, error) {
	var config PredatorConfig
	err := r.db.Where("id = ?", id).First(&config).Error
	return &config, err
}

func (r *predatorConfigRepo) GetByDiscoveryConfigID(DiscoveryConfigID int) ([]PredatorConfig, error) {
	var configs []PredatorConfig
	err := r.db.Where("discovery_config_id = ?", DiscoveryConfigID).Find(&configs).Error
	return configs, err
}

func (r *predatorConfigRepo) FindAllActiveConfig() ([]PredatorConfig, error) {
	var configs []PredatorConfig
	err := r.db.Where("active = ?", true).Find(&configs).Error
	return configs, err
}

func (r *predatorConfigRepo) FindAllActiveConfigByEmail(email string) ([]PredatorConfig, error) {
	var configs []PredatorConfig
	err := r.db.Where("active = ? AND created_by = ?", true, email).Find(&configs).Error
	return configs, err
}

func (r *predatorConfigRepo) DB() *gorm.DB {
	return r.db
}

func (r *predatorConfigRepo) BulkDeactivateByModelNames(modelNames []string, updatedBy string, discoveryConfigIds []int) error {
	if len(modelNames) == 0 || len(discoveryConfigIds) == 0 {
		return nil
	}

	return r.db.Model(&PredatorConfig{}).
		Where("model_name IN ?", modelNames).
		Where("discovery_config_id IN ?", discoveryConfigIds).
		Updates(map[string]interface{}{
			"active":     false,
			"updated_by": updatedBy,
		}).Error
}

func (r *predatorConfigRepo) FindByDiscoveryIDsAndCreatedBefore(discoveryIDs []int, daysAgo int) ([]PredatorConfig, error) {
	var configs []PredatorConfig

	cutoffDate := time.Now().AddDate(0, 0, -daysAgo)

	err := r.db.Where("discovery_config_id IN ? AND updated_at < ?", discoveryIDs, cutoffDate).
		Find(&configs).Error

	return configs, err
}

func (r *predatorConfigRepo) GetActiveModelByIds(ids []int) ([]PredatorConfig, error) {
	var configs []PredatorConfig
	err := r.db.Where("id IN ? AND active = ?", ids, true).Find(&configs).Error
	return configs, err
}

func (r *predatorConfigRepo) WithTx(tx *gorm.DB) PredatorConfigRepository {
	return &predatorConfigRepo{
		db: tx,
	}
}

func (r *predatorConfigRepo) GetByModelName(modelName string) (*PredatorConfig, error) {
	var config PredatorConfig
	err := r.db.Where("model_name = ?", modelName).First(&config).Error
	return &config, err
}

// FindByDiscoveryIDsAndAge returns active predator configs for given discovery IDs created before (now - daysAgo).
func (r *predatorConfigRepo) FindByDiscoveryIDsAndAge(discoveryConfigIds []int, daysAgo int) ([]PredatorConfig, error) {
	var configs []PredatorConfig
	if daysAgo < 0 {
		return nil, errors.New("daysAgo must be >= 0")
	}
	cutoffDate := time.Now().AddDate(0, 0, -daysAgo)

	err := r.db.Where("discovery_config_id IN ? AND created_at < ? AND active = ?",
		discoveryConfigIds, cutoffDate, true).
		Find(&configs).Error

	return configs, err
}

