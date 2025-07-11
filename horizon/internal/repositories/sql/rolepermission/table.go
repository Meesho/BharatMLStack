package rolepermission

import (
	"gorm.io/gorm"
	"time"
)

const (
	rolePermissionTable = "role_permission"
)

type RolePermission struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	Role       string `gorm:"not null"`
	Service    string `gorm:"not null"`
	ScreenType string `gorm:"not null"`
	Module     string `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (RolePermission) TableName() string {
	return rolePermissionTable
}

func (r RolePermission) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("CreatedAt", time.Now())
	return
}

func (r RolePermission) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("UpdatedAt", time.Now())
	return
}
