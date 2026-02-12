package variant_onboarding_tasks

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"gorm.io/gorm"
)

const VariantOnboardingTasksTableName = "variant_onboarding_tasks"

type VariantOnboardingTask struct {
	TaskID    int       `gorm:"primaryKey;autoIncrement" json:"task_id"`
	Entity    string    `gorm:"type:varchar(255);not null" json:"entity"`
	Model     string    `gorm:"type:varchar(255);not null" json:"model"`
	Variant   string    `gorm:"type:varchar(255);not null" json:"variant"`
	Payload   string    `gorm:"type:text" json:"payload"` // JSON string for status tracking
	Status    string    `gorm:"type:varchar(50);not null;default:'PENDING'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (VariantOnboardingTask) TableName() string {
	return VariantOnboardingTasksTableName
}

func (VariantOnboardingTask) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (VariantOnboardingTask) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
