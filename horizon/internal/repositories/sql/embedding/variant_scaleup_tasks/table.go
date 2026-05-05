package variant_scaleup_tasks

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"gorm.io/gorm"
)

const VariantScaleUpTasksTableName = "variant_scaleup_tasks"

type VariantScaleUpTask struct {
	TaskID           int       `gorm:"primaryKey;autoIncrement" json:"task_id"`
	Entity           string    `gorm:"type:varchar(255);not null" json:"entity"`
	Model            string    `gorm:"type:varchar(255);not null" json:"model"`
	Variant          string    `gorm:"type:varchar(255);not null" json:"variant"`
	VectorDBType     string    `gorm:"type:varchar(255);not null" json:"vector_db_type"`
	ScaleUpHost      string    `gorm:"type:varchar(255);not null" json:"scale_up_host"`
	TrainingDataPath string    `gorm:"type:text" json:"training_data_path"`
	Payload          string    `gorm:"type:text" json:"payload"` // JSON string for status tracking
	Status           string    `gorm:"type:varchar(50);not null;default:'PENDING'" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (VariantScaleUpTask) TableName() string {
	return VariantScaleUpTasksTableName
}

func (VariantScaleUpTask) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (VariantScaleUpTask) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
