package servicedeployableconfig

import (
	"encoding/json"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"gorm.io/gorm"
)

const ServiceDeployableTableName = "service_deployable_config"

type ServiceDeployableConfig struct {
	ID      int `gorm:"primaryKey,autoIncrement"`
	Name    string
	Host    string `gorm:"unique;not null"`
	Service string `gorm:"type:ENUM('inferflow', 'predator', 'numerix')"`
	Active  bool   `gorm:"default:false"`
	// Port                    int    `gorm:"default:8080"` // Port field for the deployable
	CreatedBy               string
	UpdatedBy               string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Config                  json.RawMessage
	MonitoringUrl           string `gorm:"default:null"`
	DeployableRunningStatus bool
	DeployableWorkFlowId    string
	DeploymentRunID         string
	DeployableHealth        string `gorm:"type:ENUM('DEPLOYMENT_REASON_ARGO_APP_HEALTH_DEGRADED', 'DEPLOYMENT_REASON_ARGO_APP_HEALTHY')"`
	WorkFlowStatus          string `gorm:"type:ENUM('WORKFLOW_COMPLETED' , 'WORKFLOW_NOT_FOUND' , 'WORKFLOW_RUNNING','WORKFLOW_FAILED' ,'WORKFLOW_NOT_STARTED' )"`
}

func (ServiceDeployableConfig) TableName() string {
	return ServiceDeployableTableName
}

func (ServiceDeployableConfig) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (ServiceDeployableConfig) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
