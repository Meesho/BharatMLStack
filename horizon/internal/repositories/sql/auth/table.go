package auth

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "users"
	createdAt = "CreatedAt"
	updatedAt = "UpdatedAt"
)

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	FirstName    string `gorm:"not null"`
	LastName     string `gorm:"not null"`
	Email        string `gorm:"unique;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null;default:user"`
	IsActive     bool   `gorm:"not null;default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string {
	return tableName
}

func (User) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (User) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}
