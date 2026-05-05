package validationlock

import (
	"time"
)

// Table represents the validation_locks table for distributed locking
type Table struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	LockKey     string    `gorm:"uniqueIndex;type:varchar(255);not null" json:"lock_key"`
	LockedBy    string    `gorm:"type:varchar(255);not null" json:"locked_by"`
	LockedAt    time.Time `gorm:"not null" json:"locked_at"`
	ExpiresAt   time.Time `gorm:"not null;index" json:"expires_at"`
	GroupID     string    `gorm:"type:varchar(255)" json:"group_id"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Table) TableName() string {
	return "validation_locks"
}
