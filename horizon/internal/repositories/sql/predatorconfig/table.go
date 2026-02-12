package predatorconfig

import (
	"encoding/json"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"gorm.io/gorm"
)

const PredatorConfigTableName = "predator_config"

type PredatorConfig struct {
	ID                int             `gorm:"primaryKey;autoIncrement"`
	DiscoveryConfigID int             `gorm:"not null"`
	ModelName         string          `gorm:"not null"`
	MetaData          json.RawMessage `gorm:"type:jsonb"`
	Active            bool            `gorm:"default:true"`
	CreatedBy         string
	UpdatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	TestResults       json.RawMessage
	HasNilData        bool `gorm:"default:false"` // Tracks if model has nil data issues
	SourceModelName   string `gorm:"column:source_model_name"`
}

func (PredatorConfig) TableName() string {
	return PredatorConfigTableName
}

func (PredatorConfig) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (PredatorConfig) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
