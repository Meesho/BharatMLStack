package validationjob

import (
	"time"
)

// Table represents the validation_jobs table for tracking async validation jobs
type Table struct {
	ID                  uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID             string     `gorm:"type:varchar(255);not null;index" json:"group_id"`
	LockID              uint       `gorm:"not null;index" json:"lock_id"`
	TestDeployableID    int        `gorm:"not null" json:"test_deployable_id"`
	ServiceName         string     `gorm:"type:varchar(255);not null" json:"service_name"`
	Status              string     `gorm:"type:varchar(50);not null;index" json:"status"` // pending, checking, completed, failed
	ValidationResult    *bool      `gorm:"default:null" json:"validation_result"`         // null=pending, true=success, false=failed
	ErrorMessage        string     `gorm:"type:text" json:"error_message"`
	RestartedAt         *time.Time `gorm:"default:null" json:"restarted_at"`
	LastHealthCheck     *time.Time `gorm:"default:null" json:"last_health_check"`
	HealthCheckCount    int        `gorm:"default:0" json:"health_check_count"`
	MaxHealthChecks     int        `gorm:"default:10" json:"max_health_checks"`
	HealthCheckInterval int        `gorm:"default:30" json:"health_check_interval"` // seconds
	CreatedAt           time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Table) TableName() string {
	return "validation_jobs"
}

// Job status constants
const (
	StatusPending   = "pending"
	StatusChecking  = "checking"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusTimeout   = "timeout"
)
