package schedule

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type ScheduleJobRepository interface {
	Create(schedule *ScheduleJob) error
}

type scheduleJobRepo struct {
	db *gorm.DB
}

// NewRepository creates a new instance of ScheduleRepository
func NewRepository(connection *infra.SQLConnection) (ScheduleJobRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &scheduleJobRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *scheduleJobRepo) Create(schedule *ScheduleJob) error {
	return r.db.Create(schedule).Error
}
