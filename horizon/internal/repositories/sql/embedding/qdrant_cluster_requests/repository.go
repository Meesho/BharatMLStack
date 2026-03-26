package qdrant_cluster_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type QdrantClusterRequestRepository interface {
	Create(req *QdrantClusterRequest) error
	GetByID(requestID int) (*QdrantClusterRequest, error)
	GetByStatus(status string) ([]QdrantClusterRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]QdrantClusterRequest, error)
}

type qdrantClusterRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (QdrantClusterRequestRepository, error) {
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

	return &qdrantClusterRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *qdrantClusterRequestRepo) Create(req *QdrantClusterRequest) error {
	return r.db.Create(req).Error
}

func (r *qdrantClusterRequestRepo) GetByID(requestID int) (*QdrantClusterRequest, error) {
	var request QdrantClusterRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *qdrantClusterRequestRepo) GetByStatus(status string) ([]QdrantClusterRequest, error) {
	var requests []QdrantClusterRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *qdrantClusterRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&QdrantClusterRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *qdrantClusterRequestRepo) GetAll() ([]QdrantClusterRequest, error) {
	var requests []QdrantClusterRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
