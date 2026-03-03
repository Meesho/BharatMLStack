package assetstate

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName  = "asset_necessity"
	resolvedAt = "ResolvedAt"
)

type Table struct {
	AssetName       string  `gorm:"primaryKey;column:asset_name;size:255"`
	Necessity       string  `gorm:"column:necessity;not null;size:20;default:active;index:idx_an_necessity"`
	ServingOverride *bool   `gorm:"column:serving_override"`
	OverrideReason  *string `gorm:"column:override_reason;type:text"`
	OverrideBy      *string `gorm:"column:override_by;size:255"`
	ResolvedAt      time.Time
	Reason          string `gorm:"column:reason;type:text;not null;default:''"`
}

func (Table) TableName() string {
	return tableName
}

func (Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(resolvedAt, time.Now())
	return
}

func (Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(resolvedAt, time.Now())
	return
}
