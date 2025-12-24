package validationjob

import (
	"errors"
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	// Create creates a new validation job
	Create(job *Table) error

	// GetByID gets a validation job by ID
	GetByID(id uint) (*Table, error)

	// GetByGroupID gets a validation job by group ID
	GetByGroupID(groupID string) (*Table, error)

	// GetPendingJobs gets all jobs that need health checking
	GetPendingJobs() ([]Table, error)

	// UpdateStatus updates the job status
	UpdateStatus(id uint, status string, errorMessage string) error

	// UpdateValidationResult updates the validation result
	UpdateValidationResult(id uint, result bool, errorMessage string) error

	// IncrementHealthCheck increments the health check count
	IncrementHealthCheck(id uint) error

	// GetJobsToCleanup gets jobs that are completed/failed and can be cleaned up
	GetJobsToCleanup(olderThan time.Duration) ([]Table, error)

	// Delete deletes a validation job
	Delete(id uint) error
}

type ValidationJobRepository struct {
	db     *gorm.DB
	dbName string
}

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

	repo := &ValidationJobRepository{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}

	// Auto-migrate the table
	if err := repo.db.AutoMigrate(&Table{}); err != nil {
		return nil, fmt.Errorf("failed to migrate validation_jobs table: %w", err)
	}

	return repo, nil
}

// Create creates a new validation job
func (r *ValidationJobRepository) Create(job *Table) error {
	result := r.db.Create(job)
	return result.Error
}

// GetByID gets a validation job by ID
func (r *ValidationJobRepository) GetByID(id uint) (*Table, error) {
	var job Table
	err := r.db.First(&job, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// GetByGroupID gets a validation job by group ID
func (r *ValidationJobRepository) GetByGroupID(groupID string) (*Table, error) {
	var job Table
	err := r.db.Where("group_id = ?", groupID).First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// GetPendingJobs gets all jobs that need health checking
func (r *ValidationJobRepository) GetPendingJobs() ([]Table, error) {
	var jobs []Table
	err := r.db.Where("status IN (?, ?)", StatusPending, StatusChecking).Find(&jobs).Error
	return jobs, err
}

// UpdateStatus updates the job status
func (r *ValidationJobRepository) UpdateStatus(id uint, status string, errorMessage string) error {
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMessage,
	}

	if status == StatusChecking {
		updates["last_health_check"] = time.Now()
	}

	result := r.db.Model(&Table{}).Where("id = ?", id).Updates(updates)
	return result.Error
}

// UpdateValidationResult updates the validation result
func (r *ValidationJobRepository) UpdateValidationResult(id uint, result bool, errorMessage string) error {
	status := StatusCompleted
	if !result {
		status = StatusFailed
	}

	updates := map[string]interface{}{
		"status":            status,
		"validation_result": result,
		"error_message":     errorMessage,
	}

	resultDB := r.db.Model(&Table{}).Where("id = ?", id).Updates(updates)
	return resultDB.Error
}

// IncrementHealthCheck increments the health check count
func (r *ValidationJobRepository) IncrementHealthCheck(id uint) error {
	result := r.db.Model(&Table{}).Where("id = ?", id).Updates(map[string]interface{}{
		"health_check_count": gorm.Expr("health_check_count + 1"),
		"last_health_check":  time.Now(),
	})
	return result.Error
}

// GetJobsToCleanup gets jobs that are completed/failed and can be cleaned up
func (r *ValidationJobRepository) GetJobsToCleanup(olderThan time.Duration) ([]Table, error) {
	var jobs []Table
	cutoffTime := time.Now().Add(-olderThan)

	err := r.db.Where("status IN (?, ?, ?) AND updated_at < ?",
		StatusCompleted, StatusFailed, StatusTimeout, cutoffTime).Find(&jobs).Error

	return jobs, err
}

// Delete deletes a validation job
func (r *ValidationJobRepository) Delete(id uint) error {
	result := r.db.Delete(&Table{}, id)
	return result.Error
}
