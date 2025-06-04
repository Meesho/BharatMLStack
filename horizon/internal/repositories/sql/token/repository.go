package token

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

// Repository defines the interface for token management operations
type Repository interface {
	SaveToken(email, token string, expiration time.Time) error
	InvalidateToken(token string) error
	IsTokenValid(token string) (bool, error)
	CleanupExpiredTokens() error
}

// TokenRepo implements Repository using gorm
type TokenRepo struct {
	db     *gorm.DB
	dbName string
}

// NewRepository creates a new token repository
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

	return &TokenRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

// SaveToken saves a new token in the database
func (t *TokenRepo) SaveToken(email, tokenStr string, expiration time.Time) error {
	userToken := &Token{
		UserEmail: email,
		Token:     tokenStr,
		ExpiresAt: expiration,
	}
	result := t.db.Create(userToken)
	return result.Error
}

// InvalidateToken removes a token from the database
func (t *TokenRepo) InvalidateToken(tokenStr string) error {
	result := t.db.Where("token = ?", tokenStr).Delete(&Token{})
	return result.Error
}

// IsTokenValid checks if a token is valid and not expired
func (t *TokenRepo) IsTokenValid(tokenStr string) (bool, error) {
	var count int64
	err := t.db.Model(&Token{}).
		Where("token = ? AND expires_at > ?", tokenStr, time.Now()).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CleanupExpiredTokens removes expired tokens from the database
func (t *TokenRepo) CleanupExpiredTokens() error {
	result := t.db.Where("expires_at < ?", time.Now()).Delete(&Token{})
	return result.Error
}
