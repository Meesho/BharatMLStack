package metadata

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type MetadataRepository interface {
	// Services
	GetAllServices() ([]Service, error)
	GetServiceByID(id uint) (*Service, error)
	GetServiceByName(name string) (*Service, error)
	CreateService(service *Service) (uint, error)
	UpdateService(id uint, service *Service) error
	DeleteService(id uint) error

	// Screen Types
	GetAllScreenTypes() ([]ScreenType, error)
	GetScreenTypesByServiceID(serviceID uint) ([]ScreenType, error)
	GetScreenTypeByID(id uint) (*ScreenType, error)
	GetScreenTypeByServiceAndName(serviceID uint, name string) (*ScreenType, error)
	CreateScreenType(screenType *ScreenType) (uint, error)
	UpdateScreenType(id uint, screenType *ScreenType) error
	DeleteScreenType(id uint) error

	// Actions
	GetAllActions() ([]Action, error)
	GetActionByID(id uint) (*Action, error)
	GetActionByName(name string) (*Action, error)
	GetActionsByIDs(ids []uint) ([]Action, error)
	CreateAction(action *Action) (uint, error)
	UpdateAction(id uint, action *Action) error
	DeleteAction(id uint) error
}

type MetadataRepo struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (MetadataRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &MetadataRepo{
		db: session.(*gorm.DB),
	}, nil
}

// ==================== Services ====================

func (r *MetadataRepo) GetAllServices() ([]Service, error) {
	var services []Service
	result := r.db.Where("is_active = ?", true).Order("display_name ASC").Find(&services)
	if result.Error != nil {
		return nil, result.Error
	}
	return services, nil
}

func (r *MetadataRepo) GetServiceByID(id uint) (*Service, error) {
	var service Service
	result := r.db.First(&service, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &service, nil
}

func (r *MetadataRepo) GetServiceByName(name string) (*Service, error) {
	var service Service
	result := r.db.Where("name = ?", name).First(&service)
	if result.Error != nil {
		return nil, result.Error
	}
	return &service, nil
}

func (r *MetadataRepo) CreateService(service *Service) (uint, error) {
	result := r.db.Create(service)
	if result.Error != nil {
		return 0, result.Error
	}
	return service.ID, nil
}

func (r *MetadataRepo) UpdateService(id uint, service *Service) error {
	result := r.db.Model(&Service{}).Where("id = ?", id).Updates(service)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("service not found")
	}
	return nil
}

func (r *MetadataRepo) DeleteService(id uint) error {
	// Soft delete by setting is_active to false
	result := r.db.Model(&Service{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("service not found")
	}
	return nil
}

// ==================== Screen Types ====================

func (r *MetadataRepo) GetAllScreenTypes() ([]ScreenType, error) {
	var screenTypes []ScreenType
	result := r.db.Where("is_active = ?", true).Preload("Service").Order("display_name ASC").Find(&screenTypes)
	if result.Error != nil {
		return nil, result.Error
	}
	return screenTypes, nil
}

func (r *MetadataRepo) GetScreenTypesByServiceID(serviceID uint) ([]ScreenType, error) {
	var screenTypes []ScreenType
	result := r.db.Where("service_id = ? AND is_active = ?", serviceID, true).Order("display_name ASC").Find(&screenTypes)
	if result.Error != nil {
		return nil, result.Error
	}
	return screenTypes, nil
}

func (r *MetadataRepo) GetScreenTypeByID(id uint) (*ScreenType, error) {
	var screenType ScreenType
	result := r.db.Preload("Service").First(&screenType, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &screenType, nil
}

func (r *MetadataRepo) GetScreenTypeByServiceAndName(serviceID uint, name string) (*ScreenType, error) {
	var screenType ScreenType
	result := r.db.Where("service_id = ? AND name = ?", serviceID, name).First(&screenType)
	if result.Error != nil {
		return nil, result.Error
	}
	return &screenType, nil
}

func (r *MetadataRepo) CreateScreenType(screenType *ScreenType) (uint, error) {
	result := r.db.Create(screenType)
	if result.Error != nil {
		return 0, result.Error
	}
	return screenType.ID, nil
}

func (r *MetadataRepo) UpdateScreenType(id uint, screenType *ScreenType) error {
	result := r.db.Model(&ScreenType{}).Where("id = ?", id).Updates(screenType)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("screen type not found")
	}
	return nil
}

func (r *MetadataRepo) DeleteScreenType(id uint) error {
	// Soft delete by setting is_active to false
	result := r.db.Model(&ScreenType{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("screen type not found")
	}
	return nil
}

// ==================== Actions ====================

func (r *MetadataRepo) GetAllActions() ([]Action, error) {
	var actions []Action
	result := r.db.Where("is_active = ?", true).Order("category ASC, display_name ASC").Find(&actions)
	if result.Error != nil {
		return nil, result.Error
	}
	return actions, nil
}

func (r *MetadataRepo) GetActionByID(id uint) (*Action, error) {
	var action Action
	result := r.db.First(&action, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &action, nil
}

func (r *MetadataRepo) GetActionByName(name string) (*Action, error) {
	var action Action
	result := r.db.Where("name = ?", name).First(&action)
	if result.Error != nil {
		return nil, result.Error
	}
	return &action, nil
}

func (r *MetadataRepo) GetActionsByIDs(ids []uint) ([]Action, error) {
	var actions []Action
	result := r.db.Where("id IN ? AND is_active = ?", ids, true).Find(&actions)
	if result.Error != nil {
		return nil, result.Error
	}
	return actions, nil
}

func (r *MetadataRepo) CreateAction(action *Action) (uint, error) {
	result := r.db.Create(action)
	if result.Error != nil {
		return 0, result.Error
	}
	return action.ID, nil
}

func (r *MetadataRepo) UpdateAction(id uint, action *Action) error {
	result := r.db.Model(&Action{}).Where("id = ?", id).Updates(action)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("action not found")
	}
	return nil
}

func (r *MetadataRepo) DeleteAction(id uint) error {
	// Soft delete by setting is_active to false
	result := r.db.Model(&Action{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("action not found")
	}
	return nil
}


