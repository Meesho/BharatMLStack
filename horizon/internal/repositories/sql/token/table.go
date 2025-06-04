package token

import (
	"gorm.io/gorm"
	"time"
)

const (
	tokenTableName = "user_tokens"
	tokenCreatedAt = "created_at"
	tokenExpiresAt = "expires_at"
)

// Token represents the structure of the user_tokens table.
type Token struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	UserEmail string    `gorm:"not null"`
	Token     string    `gorm:"unique;not null"`
	CreatedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
}

func (Token) TableName() string {
	return tokenTableName
}

func (Token) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(tokenCreatedAt, time.Now())
	return
}
