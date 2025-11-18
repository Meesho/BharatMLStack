package schedule

import (
	"gorm.io/gorm"
	"time"
)

const ScheduleTableName = "schedule_job"

type ScheduleJob struct {
	EntryDate time.Time `gorm:"primaryKey;type:date"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ScheduleJob) TableName() string {
	return ScheduleTableName
}

func (ScheduleJob) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("CreatedAt", time.Now())
	return
}

func (ScheduleJob) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn("UpdatedAt", time.Now())
	return
}
