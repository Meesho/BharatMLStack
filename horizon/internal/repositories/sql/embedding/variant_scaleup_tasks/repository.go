// horizon/internal/repositories/sql/embedding/variant_scaleup_tasks/repository.go
package variant_scaleup_tasks

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type VariantScaleUpTaskRepository interface {
	Create(task *VariantScaleUpTask) error
	GetByID(taskID int) (*VariantScaleUpTask, error)
	GetByStatus(status string) ([]VariantScaleUpTask, error)
	GetFirstPending() (*VariantScaleUpTask, error)
	GetFirstInProgress() (*VariantScaleUpTask, error)
	UpdateStatus(taskID int, status string) error
	UpdatePayload(taskID int, payload string) error
	UpdateStatusAndPayload(taskID int, status string, payload string) error
	GetAll() ([]VariantScaleUpTask, error)
}

type variantScaleUpTaskRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (VariantScaleUpTaskRepository, error) {
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

	return &variantScaleUpTaskRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *variantScaleUpTaskRepo) Create(task *VariantScaleUpTask) error {
	return r.db.Create(task).Error
}

func (r *variantScaleUpTaskRepo) GetByID(taskID int) (*VariantScaleUpTask, error) {
	var task VariantScaleUpTask
	err := r.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *variantScaleUpTaskRepo) GetByStatus(status string) ([]VariantScaleUpTask, error) {
	var tasks []VariantScaleUpTask
	err := r.db.Where("status = ?", status).Order("created_at ASC").Find(&tasks).Error
	return tasks, err
}

func (r *variantScaleUpTaskRepo) GetFirstPending() (*VariantScaleUpTask, error) {
	var task VariantScaleUpTask
	err := r.db.Where("status = ?", "PENDING").Order("created_at ASC").First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *variantScaleUpTaskRepo) GetFirstInProgress() (*VariantScaleUpTask, error) {
	var task VariantScaleUpTask
	err := r.db.Where("status = ?", "IN_PROGRESS").Order("updated_at ASC").First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *variantScaleUpTaskRepo) UpdateStatus(taskID int, status string) error {
	return r.db.Model(&VariantScaleUpTask{}).Where("task_id = ?", taskID).
		Update("status", status).Error
}

func (r *variantScaleUpTaskRepo) UpdatePayload(taskID int, payload string) error {
	return r.db.Model(&VariantScaleUpTask{}).Where("task_id = ?", taskID).
		Update("payload", payload).Error
}

func (r *variantScaleUpTaskRepo) UpdateStatusAndPayload(taskID int, status string, payload string) error {
	return r.db.Model(&VariantScaleUpTask{}).Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":  status,
			"payload": payload,
		}).Error
}

func (r *variantScaleUpTaskRepo) GetAll() ([]VariantScaleUpTask, error) {
	var tasks []VariantScaleUpTask
	err := r.db.Find(&tasks).Error
	return tasks, err
}
