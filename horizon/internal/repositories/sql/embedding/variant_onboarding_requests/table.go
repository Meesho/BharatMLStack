package variant_onboarding_requests

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"gorm.io/gorm"
)

const VariantOnboardingRequestsTableName = "variant_onboarding_requests"

type VariantOnboardingRequest struct {
	RequestID   int       `gorm:"primaryKey;autoIncrement" json:"request_id"`
	Reason      string    `gorm:"type:text;not null" json:"reason"`
	Payload     string    `gorm:"type:text;not null" json:"payload"`
	RequestType string    `gorm:"type:text;not null" json:"request_type"`
	CreatedBy   string    `gorm:"type:varchar(255);not null" json:"created_by"`
	ApprovedBy  string    `gorm:"type:varchar(255)" json:"approved_by"`
	Status      string    `gorm:"type:varchar(255);not null;default:'PENDING'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (VariantOnboardingRequest) TableName() string {
	return VariantOnboardingRequestsTableName
}

func (VariantOnboardingRequest) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (VariantOnboardingRequest) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
