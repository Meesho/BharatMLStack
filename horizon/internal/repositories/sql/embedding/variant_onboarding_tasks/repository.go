// horizon/internal/repositories/sql/embedding/variant_onboarding_tasks/repository.go
package variant_onboarding_tasks

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type VariantOnboardingTaskRepository interface {
	Create(task *VariantOnboardingTask) error
	GetByID(taskID int) (*VariantOnboardingTask, error)
	GetByStatus(status string) ([]VariantOnboardingTask, error)
	GetFirstPending() (*VariantOnboardingTask, error)
	GetFirstInProgress() (*VariantOnboardingTask, error)
	UpdateStatus(taskID int, status string) error
	UpdatePayload(taskID int, payload string) error
	UpdateStatusAndPayload(taskID int, status string, payload string) error
	GetAll() ([]VariantOnboardingTask, error)
}

type variantOnboardingTaskRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (VariantOnboardingTaskRepository, error) {
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

	return &variantOnboardingTaskRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *variantOnboardingTaskRepo) Create(task *VariantOnboardingTask) error {
	return r.db.Create(task).Error
}

func (r *variantOnboardingTaskRepo) GetByID(taskID int) (*VariantOnboardingTask, error) {
	var task VariantOnboardingTask
	err := r.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *variantOnboardingTaskRepo) GetByStatus(status string) ([]VariantOnboardingTask, error) {
	var tasks []VariantOnboardingTask
	err := r.db.Where("status = ?", status).Order("created_at ASC").Find(&tasks).Error
	return tasks, err
}

func (r *variantOnboardingTaskRepo) GetFirstPending() (*VariantOnboardingTask, error) {
	var task VariantOnboardingTask
	err := r.db.Where("status = ?", "PENDING").Order("created_at ASC").First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *variantOnboardingTaskRepo) GetFirstInProgress() (*VariantOnboardingTask, error) {
	var task VariantOnboardingTask
	err := r.db.Where("status = ?", "IN_PROGRESS").Order("updated_at ASC").First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *variantOnboardingTaskRepo) UpdateStatus(taskID int, status string) error {
	return r.db.Model(&VariantOnboardingTask{}).Where("task_id = ?", taskID).
		Update("status", status).Error
}

func (r *variantOnboardingTaskRepo) UpdatePayload(taskID int, payload string) error {
	return r.db.Model(&VariantOnboardingTask{}).Where("task_id = ?", taskID).
		Update("payload", payload).Error
}

func (r *variantOnboardingTaskRepo) UpdateStatusAndPayload(taskID int, status string, payload string) error {
	return r.db.Model(&VariantOnboardingTask{}).Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":  status,
			"payload": payload,
		}).Error
}

func (r *variantOnboardingTaskRepo) GetAll() ([]VariantOnboardingTask, error) {
	var tasks []VariantOnboardingTask
	err := r.db.Find(&tasks).Error
	return tasks, err
}
