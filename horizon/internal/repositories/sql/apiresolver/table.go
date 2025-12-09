package apiresolver

import (
	"gorm.io/gorm"
	"time"
)

const (
	apiResolversTable = "api_resolvers"
)

type ApiResolver struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	Method     string `gorm:"not null"`
	ApiPath    string `gorm:"not null"`
	ResolverFn string `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ApiResolver) TableName() string {
	return apiResolversTable
}

func (r ApiResolver) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("CreatedAt", time.Now())
	return
}

func (r ApiResolver) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("UpdatedAt", time.Now())
	return
}
