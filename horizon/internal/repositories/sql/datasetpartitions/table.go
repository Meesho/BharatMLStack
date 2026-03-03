package datasetpartitions

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "dataset_partitions"
	updatedAt = "UpdatedAt"
)

type Table struct {
	ID           uint       `gorm:"primaryKey;autoIncrement;column:id"`
	DatasetName  string     `gorm:"column:dataset_name;not null;size:255;uniqueIndex:idx_dp_unique,priority:1;index:idx_dp_dataset;index:idx_dp_ready,priority:1"`
	PartitionKey string     `gorm:"column:partition_key;not null;size:255;uniqueIndex:idx_dp_unique,priority:2"`
	DeltaVersion *int64     `gorm:"column:delta_version"`
	CommitTS     *time.Time `gorm:"column:commit_ts"`
	DQStatus     string     `gorm:"column:dq_status;not null;size:50;default:pending"`
	IsReady      bool       `gorm:"column:is_ready;not null;default:false;index:idx_dp_ready,priority:2"`
	UpdatedAt    time.Time
}

func (Table) TableName() string {
	return tableName
}

func (Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}

func (Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}
