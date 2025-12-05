package discoveryconfig

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type DiscoveryConfigRepository interface {
	Create(serviceDiscovery *DiscoveryConfig) error
	CreateTx(tx *gorm.DB, serviceDiscovery *DiscoveryConfig) error
	Update(serviceDiscovery *DiscoveryConfig) error
	UpdateTx(tx *gorm.DB, serviceDiscovery *DiscoveryConfig) error
	GetAll() ([]DiscoveryConfig, error)
	GetByToken(token string) ([]DiscoveryConfig, error)
	GetById(configId int) (*DiscoveryConfig, error)
	GetByServiceDeployableID(serviceDeployableID int) ([]DiscoveryConfig, error)
	DB() *gorm.DB
	WithTx(tx *gorm.DB) DiscoveryConfigRepository
}

type discoveryConfigRepo struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (DiscoveryConfigRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &discoveryConfigRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *discoveryConfigRepo) Create(serviceDiscovery *DiscoveryConfig) error {
	return r.db.Create(serviceDiscovery).Error
}

func (r *discoveryConfigRepo) Update(serviceDiscovery *DiscoveryConfig) error {
	return r.db.Model(serviceDiscovery).Where("id = ?", serviceDiscovery.ID).UpdateColumns(map[string]interface{}{
		"active":     serviceDiscovery.Active,
		"updated_by": serviceDiscovery.UpdatedBy,
		"updated_at": serviceDiscovery.UpdatedAt,
	}).Error
}

func (r *discoveryConfigRepo) GetAll() ([]DiscoveryConfig, error) {
	var configs []DiscoveryConfig
	err := r.db.Find(&configs).Error
	return configs, err
}

func (r *discoveryConfigRepo) GetByToken(token string) ([]DiscoveryConfig, error) {
	var configs []DiscoveryConfig
	err := r.db.Where("app_token = ?", token).Find(&configs).Error
	return configs, err
}

func (r *discoveryConfigRepo) GetById(id int) (*DiscoveryConfig, error) {
	var config DiscoveryConfig
	err := r.db.Where("id = ?", id).First(&config).Error
	return &config, err
}

func (r *discoveryConfigRepo) GetByServiceDeployableID(serviceDeployableID int) ([]DiscoveryConfig, error) {
	var configs []DiscoveryConfig
	err := r.db.Where("service_deployable_id = ?", serviceDeployableID).Find(&configs).Error
	return configs, err
}

func (r *discoveryConfigRepo) DB() *gorm.DB {
	return r.db
}

func (r *discoveryConfigRepo) CreateTx(tx *gorm.DB, serviceDiscovery *DiscoveryConfig) error {
	return tx.Create(serviceDiscovery).Error
}

func (r *discoveryConfigRepo) UpdateTx(tx *gorm.DB, serviceDiscovery *DiscoveryConfig) error {
	if err := tx.Model(serviceDiscovery).Where("id = ?", serviceDiscovery.ID).Updates(serviceDiscovery).Error; err != nil {
		return err
	}

	if !serviceDiscovery.Active {
		if err := tx.Model(serviceDiscovery).Where("id = ?", serviceDiscovery.ID).Update("active", serviceDiscovery.Active).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *discoveryConfigRepo) WithTx(tx *gorm.DB) DiscoveryConfigRepository {
	return &discoveryConfigRepo{
		db: tx,
	}
}
