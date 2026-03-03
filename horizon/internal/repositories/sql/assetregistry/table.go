package assetregistry

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "asset_specs"
	createdAt = "CreatedAt"
	updatedAt = "UpdatedAt"
)

type Table struct {
	ID           uint    `gorm:"primaryKey;autoIncrement;column:id"`
	AssetName    string  `gorm:"column:asset_name;uniqueIndex;not null;size:255"`
	AssetVersion string  `gorm:"column:asset_version;not null;size:255"`
	EntityName   string  `gorm:"column:entity_name;not null;size:255;index:idx_as_entity"`
	EntityKey    string  `gorm:"column:entity_key;not null;size:255"`
	Notebook     string  `gorm:"column:notebook;not null;size:255;index:idx_as_notebook"`
	PartitionKey string  `gorm:"column:partition_key;not null;size:100;default:ds"`
	TriggerType  string  `gorm:"column:trigger_type;not null;size:50;index:idx_as_trigger"`
	Schedule     *string `gorm:"column:schedule;size:50"`
	Serving      bool    `gorm:"column:serving;not null;default:true"`
	Incremental  bool    `gorm:"column:incremental;not null;default:false"`
	Freshness    *string `gorm:"column:freshness;size:50"`
	ChecksJSON   *string `gorm:"column:checks_json;type:text"`
	SpecJSON     string  `gorm:"column:spec_json;type:text;not null"`
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
