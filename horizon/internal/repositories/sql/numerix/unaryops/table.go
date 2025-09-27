package unaryops

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "numerix_unary_ops"
	createdAt = "CreatedAt"
	updatedAt = "UpdatedAt"
)

type Table struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	Operator   string    `gorm:"not null"`
	Parameters uint      `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time
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
