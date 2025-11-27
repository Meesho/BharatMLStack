package validationlock

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

const (
	ValidationLockKey  = "validation_process"
	DefaultLockTimeout = 30 * time.Minute // Default lock timeout
)

type Repository interface {
	// AcquireLock attempts to acquire a distributed lock
	AcquireLock(groupID string, timeout time.Duration) (*Table, error)

	// ReleaseLock releases a lock by ID
	ReleaseLock(lockID uint) error

	// ReleaseLockByKey releases a lock by key and locked_by
	ReleaseLockByKey(lockKey, lockedBy string) error

	// IsLocked checks if a lock exists and is not expired
	IsLocked(lockKey string) (bool, error)

	// CleanupExpiredLocks removes expired locks
	CleanupExpiredLocks() error

	// GetActiveLock gets the current active lock for a key
	GetActiveLock(lockKey string) (*Table, error)
}

type ValidationLockRepository struct {
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

	repo := &ValidationLockRepository{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}

	// Auto-migrate the table
	if err := repo.db.AutoMigrate(&Table{}); err != nil {
		return nil, fmt.Errorf("failed to migrate validation_locks table: %w", err)
	}

	return repo, nil
}

// AcquireLock attempts to acquire a distributed lock
func (r *ValidationLockRepository) AcquireLock(groupID string, timeout time.Duration) (*Table, error) {
	if timeout == 0 {
		timeout = DefaultLockTimeout
	}

	// Get hostname or pod name for identification
	lockedBy := getInstanceIdentifier()

	now := time.Now()
	expiresAt := now.Add(timeout)

	// First, cleanup any expired locks
	if err := r.CleanupExpiredLocks(); err != nil {
		// Log error but don't fail the lock acquisition
		fmt.Printf("Warning: failed to cleanup expired locks: %v\n", err)
	}

	lock := &Table{
		LockKey:     ValidationLockKey + "-" + groupID,
		LockedBy:    lockedBy,
		LockedAt:    now,
		ExpiresAt:   expiresAt,
		GroupID:     groupID,
		Description: fmt.Sprintf("Validation lock for group %s acquired by %s", groupID, lockedBy),
	}

	// Try to insert the lock (will fail if lock already exists due to unique constraint)
	result := r.db.Create(lock)
	if result.Error != nil {
		// Check if it's a duplicate key error (lock already exists)
		if isDuplicateKeyError(result.Error) {
			return nil, errors.New("validation lock is already held by another process")
		}
		return nil, fmt.Errorf("failed to acquire lock: %w", result.Error)
	}

	return lock, nil
}

// ReleaseLock releases a lock by ID
func (r *ValidationLockRepository) ReleaseLock(lockID uint) error {
	result := r.db.Delete(&Table{}, lockID)
	if result.Error != nil {
		return fmt.Errorf("failed to release lock: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("lock not found or already released")
	}
	return nil
}

// ReleaseLockByKey releases a lock by key and locked_by
func (r *ValidationLockRepository) ReleaseLockByKey(lockKey, lockedBy string) error {
	result := r.db.Where("lock_key = ? AND locked_by = ?", lockKey, lockedBy).Delete(&Table{})
	if result.Error != nil {
		return fmt.Errorf("failed to release lock: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("lock not found or not owned by this instance")
	}
	return nil
}

// IsLocked checks if a lock exists and is not expired
func (r *ValidationLockRepository) IsLocked(lockKey string) (bool, error) {
	var count int64
	err := r.db.Model(&Table{}).
		Where("lock_key = ? AND expires_at > ?", lockKey, time.Now()).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check lock status: %w", err)
	}

	return count > 0, nil
}

// CleanupExpiredLocks removes expired locks
func (r *ValidationLockRepository) CleanupExpiredLocks() error {
	result := r.db.Where("expires_at <= ?", time.Now()).Delete(&Table{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired locks: %w", result.Error)
	}
	return nil
}

// GetActiveLock gets the current active lock for a key
func (r *ValidationLockRepository) GetActiveLock(lockKey string) (*Table, error) {
	var lock Table
	err := r.db.Where("lock_key = ? AND expires_at > ?", lockKey, time.Now()).First(&lock).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No active lock found
		}
		return nil, fmt.Errorf("failed to get active lock: %w", err)
	}

	return &lock, nil
}

// getInstanceIdentifier returns a unique identifier for this instance
func getInstanceIdentifier() string {
	// Try to get pod name first (Kubernetes environment)
	if podName := os.Getenv("POD_NAME"); podName != "" {
		return podName
	}

	// Try to get hostname
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}

	// Fallback to a default identifier
	return "unknown-instance"
}

// isDuplicateKeyError checks if the error is a duplicate key constraint violation
func isDuplicateKeyError(err error) bool {
	// MySQL duplicate key error code is 1062
	return err != nil && (
	// Check for MySQL duplicate entry error
	fmt.Sprintf("%v", err) == "Error 1062: Duplicate entry" ||
		// Check for GORM duplicate key error patterns
		fmt.Sprintf("%v", err) == "UNIQUE constraint failed" ||
		// Check for common duplicate key error messages
		fmt.Sprintf("%v", err) == "duplicate key value violates unique constraint")
}
