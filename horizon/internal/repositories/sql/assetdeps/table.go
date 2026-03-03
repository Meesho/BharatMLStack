package assetdeps

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "asset_dependencies"
	createdAt = "CreatedAt"
	updatedAt = "UpdatedAt"
)

type Table struct {
	ID           uint   `gorm:"primaryKey;autoIncrement;column:id"`
	AssetName    string `gorm:"column:asset_name;not null;size:255;uniqueIndex:idx_asset_input;index:idx_ad_asset"`
	InputName    string `gorm:"column:input_name;not null;size:255;uniqueIndex:idx_asset_input;index:idx_ad_input"`
	InputType    string `gorm:"column:input_type;not null;size:50;default:internal"`
	PartitionKey string `gorm:"column:partition_key;not null;size:100;default:ds"`
	WindowSize   *int   `gorm:"column:window_size"`
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
