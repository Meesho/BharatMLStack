package counter

import (
	"gorm.io/gorm"
	"time"
)

const CounterTableName = "group_id_counter"

type GroupIdCounter struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	Counter   int64 `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName returns the table name for the GroupIdCounter model
func (GroupIdCounter) TableName() string {
	return CounterTableName
}

// BeforeCreate is a hook that sets the CreatedAt field before inserting a new record
func (GroupIdCounter) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("CreatedAt", time.Now())
	return
}

// BeforeUpdate is a hook that sets the UpdatedAt field before updating a record
func (GroupIdCounter) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("UpdatedAt", time.Now())
	return
}
