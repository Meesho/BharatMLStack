package store_requests

import (
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type StoreRequestRepository interface {
	Create(req *StoreRequest) error
	GetByID(requestID int) (*StoreRequest, error)
	GetByStatus(status string) ([]StoreRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]StoreRequest, error)
}

type storeRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (StoreRequestRepository, error) {
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

	return &storeRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *storeRequestRepo) Create(req *StoreRequest) error {
	return r.db.Create(req).Error
}

func (r *storeRequestRepo) GetByID(requestID int) (*StoreRequest, error) {
	var request StoreRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *storeRequestRepo) GetByStatus(status string) ([]StoreRequest, error) {
	var requests []StoreRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *storeRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	if status == "" {
		return fmt.Errorf("status cannot be empty when updating store request %d", requestID)
	}

	validStatuses := map[string]bool{
		"PENDING":            true,
		"APPROVED":           true,
		"REJECTED":           true,
		"IN_PROGRESS":        true,
		"COMPLETED":          true,
		"FAILED":             true,
		"NEEDS_MODIFICATION": true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status '%s' for store request %d. Valid statuses: PENDING, APPROVED, REJECTED, IN_PROGRESS, COMPLETED, FAILED, NEEDS_MODIFICATION", status, requestID)
	}

	return r.db.Model(&StoreRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *storeRequestRepo) GetAll() ([]StoreRequest, error) {
	var requests []StoreRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
