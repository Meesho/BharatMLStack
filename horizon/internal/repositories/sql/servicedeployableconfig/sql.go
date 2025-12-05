package servicedeployableconfig

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type ServiceDeployableRepository interface {
	Create(serviceDeployable *ServiceDeployableConfig) error
	Update(serviceDeployable *ServiceDeployableConfig) error
	DeactivateServiceDeployable(id int, deactivateBy string) error
	GetByService(service string) ([]ServiceDeployableConfig, error)
	GetById(id int) (*ServiceDeployableConfig, error)
	GetAllActive() ([]ServiceDeployableConfig, error)
	GetByWorkflowStatus(status string) ([]ServiceDeployableConfig, error)
	GetByDeployableHealth(health string) ([]ServiceDeployableConfig, error)
	GetByNameAndService(name, service string) (*ServiceDeployableConfig, error)
}

type serviceDeployableRepo struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (ServiceDeployableRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &serviceDeployableRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *serviceDeployableRepo) Create(serviceDeployable *ServiceDeployableConfig) error {
	return r.db.Create(serviceDeployable).Error
}

func (r *serviceDeployableRepo) Update(serviceDeployable *ServiceDeployableConfig) error {
	return r.db.Model(serviceDeployable).Where("id = ?", serviceDeployable.ID).Updates(serviceDeployable).Error
}

func (r *serviceDeployableRepo) DeactivateServiceDeployable(id int, deactivateBy string) error {
	return r.db.Model(&ServiceDeployableConfig{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"active":     false,
			"updated_by": deactivateBy,
			"updated_at": time.Now(),
		}).Error
}

func (r *serviceDeployableRepo) GetByService(service string) ([]ServiceDeployableConfig, error) {
	var deployables []ServiceDeployableConfig
	err := r.db.Where("service = ?", service).Find(&deployables).Error
	return deployables, err
}

func (r *serviceDeployableRepo) GetById(id int) (*ServiceDeployableConfig, error) {
	var deployable ServiceDeployableConfig
	err := r.db.Where("id = ?", id).First(&deployable).Error
	return &deployable, err
}

func (r *serviceDeployableRepo) GetAllActive() ([]ServiceDeployableConfig, error) {
	var activeDeployables []ServiceDeployableConfig
	err := r.db.Where("active = ?", true).Find(&activeDeployables).Error
	return activeDeployables, err
}

func (r *serviceDeployableRepo) GetByWorkflowStatus(status string) ([]ServiceDeployableConfig, error) {
	var results []ServiceDeployableConfig
	err := r.db.Where("work_flow_status = ?", status).Find(&results).Error
	return results, err
}

func (r *serviceDeployableRepo) GetByDeployableHealth(health string) ([]ServiceDeployableConfig, error) {
	var results []ServiceDeployableConfig
	err := r.db.Where("deployable_health = ?", health).Find(&results).Error
	return results, err
}

func (r *serviceDeployableRepo) GetByNameAndService(name, service string) (*ServiceDeployableConfig, error) {
	var deployable ServiceDeployableConfig
	err := r.db.Where("name = ? AND service = ?", name, service).First(&deployable).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &deployable, nil
}
