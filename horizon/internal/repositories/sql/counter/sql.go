package counter

import (
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupIdCounterRepository interface {
	GetAndIncrementCounter(id int) (uint, error)
}

// groupIdCounterRepo implements the GroupIdCounterRepository interface
type groupIdCounterRepo struct {
	db *gorm.DB
}

func NewCounterRepository(connection *infra.SQLConnection) (GroupIdCounterRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &groupIdCounterRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *groupIdCounterRepo) GetAndIncrementCounter(id int) (uint, error) {
	var counter GroupIdCounter

	tx := r.db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&counter).Error

	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to fetch group ID counter: %w", err)
	}

	counter.Counter++

	if err := tx.Model(&counter).Where("id = ?", counter.ID).Update("counter", counter.Counter).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to update group ID counter: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return uint(counter.Counter), nil
}
