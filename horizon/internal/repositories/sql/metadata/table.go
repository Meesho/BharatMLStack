package metadata

import (
	"time"

	"gorm.io/gorm"
)

const (
	servicesTable    = "services"
	screenTypesTable = "screen_types"
	actionsTable     = "actions"
	createdAt        = "CreatedAt"
	updatedAt        = "UpdatedAt"
)

// Service represents a service in the system
type Service struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"type:varchar(255);not null;uniqueIndex:unique_name"`
	DisplayName string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:text"`
	IsActive    bool   `gorm:"default:true;index:idx_is_active"`
	CreatedBy   *uint  `gorm:"foreignKey:CreatedBy"`
	UpdatedBy   *uint  `gorm:"foreignKey:UpdatedBy"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Service) TableName() string {
	return servicesTable
}

func (s *Service) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (s *Service) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}

// ScreenType represents a screen type belonging to a service
type ScreenType struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"`
	ServiceID   uint    `gorm:"not null;index:idx_service_active;uniqueIndex:unique_service_screen"`
	Service     Service `gorm:"foreignKey:ServiceID;constraint:OnDelete:CASCADE"`
	Name        string  `gorm:"type:varchar(255);not null;uniqueIndex:unique_service_screen"`
	DisplayName string  `gorm:"type:varchar(255);not null"`
	Description string  `gorm:"type:text"`
	IsActive    bool    `gorm:"default:true;index:idx_service_active"`
	CreatedBy   *uint   `gorm:"foreignKey:CreatedBy"`
	UpdatedBy   *uint   `gorm:"foreignKey:UpdatedBy"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ScreenType) TableName() string {
	return screenTypesTable
}

func (st *ScreenType) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (st *ScreenType) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}

// Action represents an action that can be performed
type Action struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"type:varchar(255);not null;uniqueIndex:unique_name"`
	DisplayName string `gorm:"type:varchar(255);not null"`
	Category    string `gorm:"type:varchar(50);index:idx_category"` // 'crud', 'approval', 'testing', 'management'
	Description string `gorm:"type:text"`
	IsActive    bool   `gorm:"default:true;index:idx_is_active"`
	CreatedBy   *uint  `gorm:"foreignKey:CreatedBy"`
	UpdatedBy   *uint  `gorm:"foreignKey:UpdatedBy"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Action) TableName() string {
	return actionsTable
}

func (a *Action) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (a *Action) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}


