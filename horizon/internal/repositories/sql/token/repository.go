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
	SaveRefreshToken(email, refreshToken string, expiration time.Time) error
	GetRefreshToken(refreshToken string) (*Token, error)
	InvalidateToken(token string) error
	InvalidateRefreshToken(refreshToken string) error
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

// SaveToken saves a new access token in the database
func (t *TokenRepo) SaveToken(email, tokenStr string, expiration time.Time) error {
	userToken := &Token{
		UserEmail: email,
		Token:     tokenStr,
		TokenType: "access", // Explicitly set token type to access
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

// IsTokenValid checks if an access token is valid and not expired
// Only validates access tokens (not refresh tokens)
func (t *TokenRepo) IsTokenValid(tokenStr string) (bool, error) {
	var count int64
	err := t.db.Model(&Token{}).
		Where("token = ? AND token_type = ? AND expires_at > ?", tokenStr, "access", time.Now()).
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

// SaveRefreshToken saves a refresh token in the database
// Uses Token field to store the refresh token value, TokenType distinguishes it from access tokens
func (t *TokenRepo) SaveRefreshToken(email, refreshToken string, expiration time.Time) error {
	userToken := &Token{
		UserEmail:    email,
		Token:        refreshToken, // Store refresh token in Token field
		RefreshToken: nil,          // Not needed - TokenType distinguishes token types
		TokenType:    "refresh",
		ExpiresAt:    expiration,
	}
	result := t.db.Create(userToken)
	return result.Error
}

// GetRefreshToken retrieves a refresh token from the database
// Queries by token value and token_type to find the refresh token
func (t *TokenRepo) GetRefreshToken(refreshToken string) (*Token, error) {
	var token Token
	result := t.db.Where("token = ? AND token_type = ? AND expires_at > ?", refreshToken, "refresh", time.Now()).First(&token)
	return &token, result.Error
}

// InvalidateRefreshToken removes a refresh token from the database
// Queries by token value and token_type to find the refresh token
func (t *TokenRepo) InvalidateRefreshToken(refreshToken string) error {
	result := t.db.Where("token = ? AND token_type = ?", refreshToken, "refresh").Delete(&Token{})
	return result.Error
}
