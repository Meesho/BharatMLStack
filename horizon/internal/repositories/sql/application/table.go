package application

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"

	"gorm.io/gorm"
)

const ApplicationTableName = "application_config"

type Application struct {
	AppToken  string `gorm:"primaryKey"`
	Bu        string `gorm:"not null"`
	Team      string `gorm:"not null"`
	Service   string `gorm:"not null"`
	Active    bool   `gorm:"not null"`
	CreatedBy string `gorm:"not null"`
	UpdatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Application) TableName() string {
	return ApplicationTableName
}

func (Application) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (Application) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
