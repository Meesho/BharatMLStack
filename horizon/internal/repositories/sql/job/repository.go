package job

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	GetAll() ([]Table, error)
	Create(table *Table) (uint, error)
	Update(table *Table) error
	GetById(id int) (*Table, error)
	GetAllByUserId(userId string) ([]Table, error)
}

// Repository implements RepositoryInterface using a generic repository
type Job struct {
	db     *gorm.DB
	dbName string
}

// NewRepository creates a new job repository
func NewRepository(connection *infra.SQLConnection) (Repository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}
	meta, err := connection.GetMeta()
	if err != nil {
		return nil, err
	}
	dbName := meta["db_name"].(string)

	return &Job{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

// GetAll retrieves all job records from the database.
func (j *Job) GetAll() ([]Table, error) {
	var jobs []Table
	result := j.db.Find(&jobs)
	return jobs, result.Error
}

// GetById retrieves a job by its ID.
func (j *Job) GetById(id int) (*Table, error) {
	var job Table
	result := j.db.Where("request_id = ?", id).First(&job)
	return &job, result.Error
}

// GetAllByUserId retrieves all jobs created by a specific user
func (j *Job) GetAllByUserId(userId string) ([]Table, error) {
	var jobs []Table
	result := j.db.Where("created_by = ?", userId).Find(&jobs)
	return jobs, result.Error
}

// Create adds a new job to the database.
func (j *Job) Create(table *Table) (uint, error) {
	result := j.db.Create(table)
	if result.Error != nil {
		return 0, result.Error
	}
	return table.RequestId, nil
}

// Update updates a job's information in the database.
func (j *Job) Update(table *Table) error {
	result := j.db.Model(table).Where("request_id = ?", table.RequestId).Updates(table)
	return result.Error
}
