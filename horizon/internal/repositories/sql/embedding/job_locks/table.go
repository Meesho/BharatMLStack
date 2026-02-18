package job_locks

import (
	"time"
)

const JobLocksTableName = "skye_job_locks"

type SkyeJobLock struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	JobKey    string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"job_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SkyeJobLock) TableName() string {
	return JobLocksTableName
}
