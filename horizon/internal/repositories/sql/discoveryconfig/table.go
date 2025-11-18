package discoveryconfig

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"gorm.io/gorm"
)

const DiscoveryConfigTableName = "discovery_config"

type DiscoveryConfig struct {
	ID                          int    `gorm:"primaryKey;autoIncrement"`
	AppToken                    string `gorm:"foreignKey:AppToken"`
	ServiceDeployableID         int    `gorm:"foreignKey:ServiceDeployableID"`
	CircuitBreakerID            *int   // Nullable foreign key
	ServiceConnectionID         int    `gorm:"foreignKey:ServiceConnectionID"`
	RouteServiceConnectionID    *int   // Nullable foreign key
	RouteServiceDeployableID    *int   // Nullable foreign key
	RoutePercent                *int   // Nullable
	Active                      bool   `gorm:"default:true"`
	DefaultResponse             string `gorm:"size:5000"`
	DefaultResponsePercent      *int   `gorm:""`
	DefaultResponseEnabledForCB bool   `gorm:"default:false;column:default_response_enabled_forCB"`
	CreatedBy                   string
	UpdatedBy                   string
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

func (DiscoveryConfig) TableName() string {
	return DiscoveryConfigTableName
}

func (DiscoveryConfig) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (DiscoveryConfig) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
