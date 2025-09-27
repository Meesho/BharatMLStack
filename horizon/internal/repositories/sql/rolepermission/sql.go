package rolepermission

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	CheckPermission(role, service, screenType, module string) (bool, error)
	CreatePermission(permission *RolePermission) (uint, error)
	GetPermissionsByRole(role string) ([]RolePermission, error)
}

type RolePermissionRepository struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (*RolePermissionRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}
	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &RolePermissionRepository{
		db: session.(*gorm.DB),
	}, nil
}

// CheckPermission returns true if the given role has access to the screen/module under a specific service
func (store *RolePermissionRepository) CheckPermission(role, service, screenType, module string) (bool, error) {
	var perm RolePermission
	result := store.db.
		Where("role = ? AND service = ? AND screen_type = ? AND module = ?", role, service, screenType, module).
		First(&perm)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

// CreatePermission adds a role permission entry
func (store *RolePermissionRepository) CreatePermission(permission *RolePermission) (uint, error) {
	result := store.db.Create(permission)
	if result.Error != nil {
		return 0, result.Error
	}
	return permission.ID, nil
}

// GetPermissionsByRole retrieves all permissions for a given role
func (store *RolePermissionRepository) GetPermissionsByRole(role string) ([]RolePermission, error) {
	var permissions []RolePermission
	result := store.db.Where("role = ?", role).Find(&permissions)
	return permissions, result.Error
}
