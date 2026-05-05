package deployablemetadata

import (
	"gorm.io/gorm"
	"time"
)

const DeployableMetadataTableName = "deployable_metadata"

type DeployableMetadata struct {
	ID        int       `gorm:"primaryKey"`
	Key       string    `gorm:"not null"`
	Value     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	Active    bool      `gorm:"default:true"`
}

func (DeployableMetadata) TableName() string {
	return DeployableMetadataTableName
}

func (DeployableMetadata) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("CreatedAt", time.Now())
	return
}

func (DeployableMetadata) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("UpdatedAt", time.Now())
	return
}
