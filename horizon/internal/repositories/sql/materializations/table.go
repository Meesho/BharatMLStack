package materializations

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "materializations"
	createdAt = "CreatedAt"
)

type Table struct {
	ID               uint   `gorm:"primaryKey;autoIncrement;column:id"`
	AssetName        string `gorm:"column:asset_name;not null;size:255;uniqueIndex:idx_mat_unique,priority:1;index:idx_mat_asset_part,priority:1"`
	PartitionKey     string `gorm:"column:partition_key;not null;size:255;uniqueIndex:idx_mat_unique,priority:2;index:idx_mat_asset_part,priority:2"`
	ComputeKey       string `gorm:"column:compute_key;not null;size:255;uniqueIndex:idx_mat_unique,priority:3;index:idx_mat_compute_key"`
	ArtifactPath     string `gorm:"column:artifact_path;type:text;not null"`
	InputFingerprint string `gorm:"column:input_fingerprint;type:text;not null"`
	Status           string `gorm:"column:status;not null;size:50"`
	CreatedAt        time.Time
}

func (Table) TableName() string {
	return tableName
}

func (Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}
