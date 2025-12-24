package predatorrequest

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

// PredatorRequestRepository defines the interface for CRUD operations
type PredatorRequestRepository interface {
	GetAll() ([]PredatorRequest, error)
	GetByID(id uint) (*PredatorRequest, error)
	GetAllByEmail(email string) ([]PredatorRequest, error)
	Create(req *PredatorRequest) error
	Update(req *PredatorRequest) error
	Delete(id uint) error
	DB() *gorm.DB
	UpdateStatusAndStage(tx *gorm.DB, requestModel *PredatorRequest) error
	ActiveModelRequestExistForRequestType(modelNames []string, requestType string) (bool, error)
	WithTx(tx *gorm.DB) PredatorRequestRepository
	UpdateMany(reqs []PredatorRequest) error
	CreateMany(reqs []PredatorRequest) error
	GetAllByGroupID(groupID uint) ([]PredatorRequest, error)
}

// predatorRequestRepo implements the repository interface
type predatorRequestRepo struct {
	db *gorm.DB
}

// NewRepository creates a new PredatorRequest repository
func NewRepository(connection *infra.SQLConnection) (PredatorRequestRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &predatorRequestRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *predatorRequestRepo) GetAll() ([]PredatorRequest, error) {
	var requests []PredatorRequest
	err := r.db.Find(&requests).Error
	return requests, err
}

func (r *predatorRequestRepo) GetByID(id uint) (*PredatorRequest, error) {
	var req PredatorRequest
	err := r.db.Where("request_id = ?", id).First(&req).Error
	return &req, err
}

func (r *predatorRequestRepo) Create(req *PredatorRequest) error {
	return r.db.Create(req).Error
}

func (r *predatorRequestRepo) Update(req *PredatorRequest) error {
	return r.db.Model(req).Where("request_id = ?", req.RequestID).Select("*").Updates(req).Error
}

func (r *predatorRequestRepo) Delete(id uint) error {
	return r.db.Where("request_id = ?", id).Delete(&PredatorRequest{}).Error
}

func (r *predatorRequestRepo) GetAllByEmail(email string) ([]PredatorRequest, error) {
	var requests []PredatorRequest
	err := r.db.
		Where("created_by = ?", email).
		Find(&requests).Error
	return requests, err
}

func (r *predatorRequestRepo) DB() *gorm.DB {
	return r.db
}

func (r *predatorRequestRepo) UpdateStatusAndStage(tx *gorm.DB, requestModel *PredatorRequest) error {
	return tx.Model(&requestModel).Where("request_id = ?", requestModel.RequestID).Updates(requestModel).Error
}

func (r *predatorRequestRepo) CreateMany(reqs []PredatorRequest) error {
	if len(reqs) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, req := range reqs {
			if err := tx.Create(&req).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *predatorRequestRepo) ActiveModelRequestExistForRequestType(modelNames []string, requestType string) (bool, error) {
	// Query the database to check for active models
	var count int64
	err := r.db.Model(&PredatorRequest{}).
		Where("model_name IN (?)", modelNames).
		Where("request_type = ?", requestType).
		Where("active = ?", true).
		Count(&count).Error

	if err != nil {
		return false, err // Error in querying
	}

	// Return true if there are any active models, otherwise false
	return count > 0, nil
}

func (r *predatorRequestRepo) UpdateMany(reqs []PredatorRequest) error {
	if len(reqs) == 0 {
		return nil
	}

	tx := r.db.Begin()
	for _, req := range reqs {
		err := tx.Model(&req).Where("request_id = ?", req.RequestID).Updates(map[string]interface{}{
			"status":        req.Status,
			"request_stage": req.RequestStage,
			"reviewer":      req.Reviewer,
			"updated_by":    req.UpdatedBy,
			"updated_at":    req.UpdatedAt,
			"active":        req.Active,
			"reject_reason": req.RejectReason,
		}).Error

		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (r *predatorRequestRepo) GetAllByGroupID(groupID uint) ([]PredatorRequest, error) {
	var requests []PredatorRequest
	err := r.db.
		Where("group_id = ?", groupID).Where("active = ?", true).
		Find(&requests).Error
	return requests, err
}

func (r *predatorRequestRepo) WithTx(tx *gorm.DB) PredatorRequestRepository {
	return &predatorRequestRepo{
		db: tx,
	}
}
