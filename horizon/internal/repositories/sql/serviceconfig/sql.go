package serviceconfig

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type ServiceConfigRepository interface {
	Create(serviceConfig *ServiceConfig) error
	Update(serviceConfig *ServiceConfig) error
	GetByID(id int) (*ServiceConfig, error)
	GetByName(name string) (*ServiceConfig, error)
	Delete(id int) error
	ListAll() ([]ServiceConfig, error)
}

type serviceConfigRepo struct {
	db *gorm.DB
}

func (r *serviceConfigRepo) GetByName(name string) (*ServiceConfig, error) {
	var config ServiceConfig
	err := r.db.Where("service_name = ?", name).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func NewRepository(connection *infra.SQLConnection) (ServiceConfigRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &serviceConfigRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *serviceConfigRepo) Create(serviceConfig *ServiceConfig) error {
	serviceConfig.CreatedAt = time.Now()
	serviceConfig.UpdatedAt = time.Now()
	return r.db.Create(serviceConfig).Error
}

func (r *serviceConfigRepo) Update(serviceConfig *ServiceConfig) error {
	serviceConfig.UpdatedAt = time.Now()
	return r.db.Model(serviceConfig).Where("id = ?", serviceConfig.ID).Updates(serviceConfig).Error
}

func (r *serviceConfigRepo) GetByID(id int) (*ServiceConfig, error) {
	var config ServiceConfig
	err := r.db.Where("id = ?", id).First(&config).Error
	return &config, err
}

func (r *serviceConfigRepo) Delete(id int) error {
	return r.db.Where("id = ?", id).Delete(&ServiceConfig{}).Error
}

func (r *serviceConfigRepo) ListAll() ([]ServiceConfig, error) {
	var configs []ServiceConfig
	err := r.db.Find(&configs).Error
	return configs, err
}
