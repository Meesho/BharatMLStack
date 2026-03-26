package permissions

import (
	"time"

	"gorm.io/gorm"
)

const (
	permissionsTable = "permissions"
	createdAt        = "CreatedAt"
	updatedAt        = "UpdatedAt"
)

type Permission struct {
	ID             uint   `gorm:"primaryKey;autoIncrement"`
	Role           string `gorm:"type:enum('super_admin','admin','user');not null;index:idx_role"`
	ServiceID      uint   `gorm:"not null;index:idx_service_screen;index:idx_role_service_screen"`
	ScreenTypeID   uint   `gorm:"not null;index:idx_service_screen;index:idx_role_service_screen"`
	AllowedActions string `gorm:"type:json;not null"` // JSON array of action IDs: [1, 2, 3]
	CreatedBy      uint   `gorm:"not null;foreignKey:CreatedBy"`
	UpdatedBy      uint   `gorm:"not null;foreignKey:UpdatedBy"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Permission) TableName() string {
	return permissionsTable
}

func (Permission) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (Permission) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}
