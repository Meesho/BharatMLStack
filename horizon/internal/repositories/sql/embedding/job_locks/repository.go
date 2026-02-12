package job_locks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type JobLocksRepository interface {
	EnsureRow(jobKey string) error
	AcquireLockNowait(ctx context.Context, jobKey string) error
	ReleaseLock(ctx context.Context, jobKey string) error
}

type jobLocksRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (JobLocksRepository, error) {
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

	return &jobLocksRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *jobLocksRepo) EnsureRow(jobKey string) error {
	// Try to create; ignore duplicate key errors by checking existence when create fails
	lock := SkyeJobLock{JobKey: jobKey}
	if err := r.db.Create(&lock).Error; err != nil {
		// If record exists, ensure by querying
		var existing SkyeJobLock
		if err2 := r.db.Where("job_key = ?", jobKey).First(&existing).Error; err2 != nil {
			return fmt.Errorf("ensure row create failed: %w", err)
		}
	}
	return nil
}

func (r *jobLocksRepo) AcquireLockNowait(ctx context.Context, jobKey string) error {
	// Get underlying *sql.DB from gorm DB
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	// Acquire a dedicated connection so the session-level lock is tied to it
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get dedicated conn: %w", err)
	}
	defer conn.Close()

	// Use a deterministic lock name. Prefix with DB name to avoid cross-db collisions
	lockName := fmt.Sprintf("%s:%s", r.dbName, jobKey)

	// GET_LOCK returns 1 (success), 0 (timeout), or NULL (error). We use timeout 0 to fail-fast.
	var res sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", lockName).Scan(&res); err != nil {
		return fmt.Errorf("get_lock query failed: %w", err)
	}
	if !res.Valid || res.Int64 != 1 {
		// failed to acquire lock immediately
		return fmt.Errorf("unable to get lock for %s", jobKey)
	}

	return nil
}

func (r *jobLocksRepo) ReleaseLock(ctx context.Context, jobKey string) error {
	// Get underlying *sql.DB from gorm DB
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	// Acquire a dedicated connection so the session-level lock is tied to it
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get dedicated conn: %w", err)
	}
	defer conn.Close()

	// Use a deterministic lock name. Prefix with DB name to avoid cross-db collisions
	lockName := fmt.Sprintf("%s:%s", r.dbName, jobKey)

	// RELEASE_LOCK returns 1 (success), 0 (timeout), or NULL (error). We use timeout 0 to fail-fast.
	var res sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", lockName).Scan(&res); err != nil {
		return fmt.Errorf("release_lock query failed: %w", err)
	}
	if !res.Valid || res.Int64 != 1 {
		return fmt.Errorf("unable to release lock for %s", jobKey)
	}
	return nil
}
