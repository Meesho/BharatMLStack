package config

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	discovery "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	"gorm.io/gorm"
)

const (
	tableName = "inferflow_config"
)

type Table struct {
	ID              uint                      `gorm:"primaryKey;autoIncrement"`
	DiscoveryID     int                       `gorm:"not null"`
	Discovery       discovery.DiscoveryConfig `gorm:"foreignKey:ID"`
	ConfigID        string                    `gorm:"unique"`
	ConfigValue     inferflow.InferflowConfig `gorm:"type:json;not null"`
	DefaultResponse inferflow.DefaultResponse `gorm:"type:json"`
	CreatedBy       string                    `gorm:"not null"`
	UpdatedBy       string
	Active          bool      `gorm:"not null"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time
	TestResults     inferflow.TestResults `gorm:"type:json"`
}

func (Table) TableName() string {
	return tableName
}

func (Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
