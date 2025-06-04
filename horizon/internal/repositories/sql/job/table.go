package job

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "job"
	createdAt = "CreatedAt"
	updatedAt = "UpdatedAt"
)

type Table struct {
	RequestId    uint   `gorm:"primaryKey;autoIncrement"`
	JobId        string `gorm:"not null"`
	Payload      string `gorm:"not null"`
	CreatedBy    string `gorm:"not null"`
	ApprovedBy   string `gorm:"not null"`
	Status       string `gorm:"not null"`
	Service      string `gorm:"not null"`
	RejectReason string `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Table) TableName() string {
	return tableName
}

func (Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}
