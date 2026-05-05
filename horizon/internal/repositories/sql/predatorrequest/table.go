package predatorrequest

import (
	"time"

	"gorm.io/gorm"
)

const PredatorRequestTableName = "predator_requests"

type PredatorRequest struct {
	RequestID    uint   `gorm:"primaryKey;autoIncrement"`
	GroupId      uint   `gorm:"not null"`
	ModelName    string `gorm:"not null"`
	Payload      string `gorm:"type:json;not null"`
	CreatedBy    string `gorm:"not null"`
	UpdatedBy    string
	Reviewer     string
	RequestStage string `gorm:"type:enum('Clone To Bucket','Restart Deployable','DB Population','Pending','Request Payload Error');default:'Pending'"`
	RequestType  string `gorm:"type:enum('Onboard','ScaleUp','Promote','Delete' , 'Edit');not null"`
	Status       string `gorm:"type:enum('Pending Approval','Approved','Rejected','Cancelled','Failed' , 'In Progress', '');default:'Pending Approval'"`
	Active       bool   `gorm:"default:true"`
	RejectReason string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IsValid      bool `gorm:"default:false"`
}

func (PredatorRequest) TableName() string {
	return PredatorRequestTableName
}

func (PredatorRequest) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("CreatedAt", time.Now())
	return
}

func (PredatorRequest) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("UpdatedAt", time.Now())
	return
}
