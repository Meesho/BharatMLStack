package serviceconfig

import (
	"encoding/json"
	"gorm.io/gorm"
	"time"
)

const ServiceConfigTableName = "service_config"

type ServiceConfig struct {
	ID             int    `gorm:"primaryKey"`
	ServiceName    string `gorm:"not null"`
	PrimaryOwner   string
	SecondaryOwner string
	RepoName       string
	BranchName     string
	HealthCheck    string
	AppPort        int
	Team           string
	BU             string
	PriorityV2     string
	Module         string
	AppType        string
	IngressClass   string
	BuildNo        string
	Config         json.RawMessage `gorm:"type:json"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (ServiceConfig) TableName() string {
	return ServiceConfigTableName
}

func (ServiceConfig) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("CreatedAt", time.Now())
	return
}

func (ServiceConfig) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("UpdatedAt", time.Now())
	return
}
