package variant_promotion_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type VariantPromotionRequestRepository interface {
	Create(req *VariantPromotionRequest) error
	GetByID(requestID int) (*VariantPromotionRequest, error)
	GetByStatus(status string) ([]VariantPromotionRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]VariantPromotionRequest, error)
}

type variantPromotionRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (VariantPromotionRequestRepository, error) {
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

	return &variantPromotionRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *variantPromotionRequestRepo) Create(req *VariantPromotionRequest) error {
	return r.db.Create(req).Error
}

func (r *variantPromotionRequestRepo) GetByID(requestID int) (*VariantPromotionRequest, error) {
	var request VariantPromotionRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *variantPromotionRequestRepo) GetByStatus(status string) ([]VariantPromotionRequest, error) {
	var requests []VariantPromotionRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *variantPromotionRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&VariantPromotionRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *variantPromotionRequestRepo) GetAll() ([]VariantPromotionRequest, error) {
	var requests []VariantPromotionRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
