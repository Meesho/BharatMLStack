package job_frequency_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type JobFrequencyRequestRepository interface {
	Create(req *JobFrequencyRequest) error
	GetByID(requestID int) (*JobFrequencyRequest, error)
	GetByStatus(status string) ([]JobFrequencyRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]JobFrequencyRequest, error)
}

type jobFrequencyRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (JobFrequencyRequestRepository, error) {
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

	return &jobFrequencyRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *jobFrequencyRequestRepo) Create(req *JobFrequencyRequest) error {
	return r.db.Create(req).Error
}

func (r *jobFrequencyRequestRepo) GetByID(requestID int) (*JobFrequencyRequest, error) {
	var request JobFrequencyRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *jobFrequencyRequestRepo) GetByStatus(status string) ([]JobFrequencyRequest, error) {
	var requests []JobFrequencyRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *jobFrequencyRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&JobFrequencyRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *jobFrequencyRequestRepo) GetAll() ([]JobFrequencyRequest, error) {
	var requests []JobFrequencyRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
