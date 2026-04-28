package permissions

import (
	"encoding/json"
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	GetPermission(role string, serviceID, screenTypeID uint) (*Permission, error)
	CheckPermission(role string, serviceID, screenTypeID uint, actionID uint) (bool, error)
	CreatePermission(permission *Permission) (uint, error)
	UpdatePermission(id uint, permission *Permission) error
	DeletePermission(id uint) error
	GetPermissionsByRole(role string) ([]Permission, error)
	GetAllPermissions() ([]Permission, error)
	BulkUpdatePermissionsByRole(role string, permissionList []Permission) error
}

type PermissionRepository struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (Repository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &PermissionRepository{
		db: session.(*gorm.DB),
	}, nil
}

// GetPermission retrieves a permission by role, service_id, and screen_type_id
func (p *PermissionRepository) GetPermission(role string, serviceID, screenTypeID uint) (*Permission, error) {
	var permission Permission
	result := p.db.Where("role = ? AND service_id = ? AND screen_type_id = ?", role, serviceID, screenTypeID).First(&permission)
	if result.Error != nil {
		return nil, result.Error
	}
	return &permission, nil
}

// CheckPermission checks if an action is allowed for a given role, service_id, screen_type_id, and action_id
func (p *PermissionRepository) CheckPermission(role string, serviceID, screenTypeID uint, actionID uint) (bool, error) {
	permission, err := p.GetPermission(role, serviceID, screenTypeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	// Parse allowed_actions JSON array (array of action IDs)
	var allowedActionIDs []uint
	if err := json.Unmarshal([]byte(permission.AllowedActions), &allowedActionIDs); err != nil {
		return false, err
	}

	// Check if action_id is in allowed_actions
	for _, allowedActionID := range allowedActionIDs {
		if allowedActionID == actionID {
			return true, nil
		}
	}

	return false, nil
}

// CreatePermission creates a new permission
func (p *PermissionRepository) CreatePermission(permission *Permission) (uint, error) {
	result := p.db.Create(permission)
	if result.Error != nil {
		return 0, result.Error
	}
	return permission.ID, nil
}

// UpdatePermission updates an existing permission
func (p *PermissionRepository) UpdatePermission(id uint, permission *Permission) error {
	result := p.db.Model(&Permission{}).Where("id = ?", id).Updates(permission)
	return result.Error
}

// DeletePermission deletes a permission
func (p *PermissionRepository) DeletePermission(id uint) error {
	result := p.db.Where("id = ?", id).Delete(&Permission{})
	return result.Error
}

// GetPermissionsByRole retrieves all permissions for a given role
func (p *PermissionRepository) GetPermissionsByRole(role string) ([]Permission, error) {
	var permissions []Permission
	result := p.db.Where("role = ?", role).Find(&permissions)
	return permissions, result.Error
}

// GetAllPermissions retrieves all permissions
func (p *PermissionRepository) GetAllPermissions() ([]Permission, error) {
	var permissions []Permission
	result := p.db.Find(&permissions)
	return permissions, result.Error
}

// BulkUpdatePermissionsByRole updates all permissions for a role
func (p *PermissionRepository) BulkUpdatePermissionsByRole(role string, permissionList []Permission) error {
	// Delete existing permissions for the role
	if err := p.db.Where("role = ?", role).Delete(&Permission{}).Error; err != nil {
		return err
	}

	// Create new permissions
	for _, permission := range permissionList {
		permission.Role = role
		if err := p.db.Create(&permission).Error; err != nil {
			return err
		}
	}

	return nil
}


