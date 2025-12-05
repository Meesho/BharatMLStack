package request

import (
	"time"

	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	"gorm.io/gorm"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
)

const (
	tableName = "inferflow_request"
)

type Table struct {
	RequestID    uint              `gorm:"primaryKey;autoIncrement"`
	ConfigID     string            `gorm:"not null"`
	Payload      inferflow.Payload `gorm:"type:json;not null"`
	CreatedBy    string            `gorm:"not null"`
	Version      int               `gorm:"not null;default:1"`
	UpdatedBy    string
	Reviewer     string
	Status       string `gorm:"not null"`
	RequestType  string `gorm:"not null"`
	RejectReason string
	Active       bool      `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time
}

func (Table) TableName() string {
	return tableName
}

func (t *Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (t *Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
