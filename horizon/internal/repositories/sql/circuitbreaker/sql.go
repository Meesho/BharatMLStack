package circuitbreaker

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type CBRepository interface {
	GetAll() ([]CircuitBreaker, error)
	Create(cb *CircuitBreaker) error
	Update(cb *CircuitBreaker) error
	GetById(id int) (*CircuitBreaker, error)
}

type cbRepo struct {
	db *gorm.DB
}

func NewCBRepository(connection *infra.SQLConnection) (CBRepository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &cbRepo{
		db: session.(*gorm.DB),
	}, nil
}

func (r *cbRepo) GetAll() ([]CircuitBreaker, error) {
	var circuitBreakers []CircuitBreaker
	err := r.db.Find(&circuitBreakers).Error
	return circuitBreakers, err
}

func (r *cbRepo) GetById(id int) (*CircuitBreaker, error) {
	var cb CircuitBreaker
	err := r.db.Where("id = ?", id).First(&cb).Error
	return &cb, err
}

func (r *cbRepo) Create(cb *CircuitBreaker) error {
	return r.db.Create(cb).Error
}

func (r *cbRepo) Update(cb *CircuitBreaker) error {
	return r.db.Model(cb).Where("id = ?", cb.ID).Updates(cb).Error
}
