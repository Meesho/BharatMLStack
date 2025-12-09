package numerix_request

import (
	"encoding/json"
	"errors"
	"time"

	"database/sql/driver"

	"gorm.io/gorm"
)

const (
	tableName = "numerix_request"
	createdAt = "CreatedAt"
	updatedAt = "UpdatedAt"
)

type Expression struct {
	InfixExpression   string `json:"infix_expression"`
	PostfixExpression string `json:"postfix_expression"`
}

type RequestExpression struct {
	Expression Expression `json:"expression"`
}

func (r *RequestExpression) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for RequestExpression")
	}
	return json.Unmarshal(bytes, r)
}

func (r RequestExpression) Value() (driver.Value, error) {
	return json.Marshal(r)
}

type Table struct {
	RequestID    uint   `gorm:"primaryKey;autoIncrement"`
	ConfigID     uint   `gorm:"not null"`
	CreatedBy    string `gorm:"not null"`
	UpdatedBy    string
	Reviewer     string `gorm:"not null"`
	Status       string `gorm:"not null"`
	RequestType  string `gorm:"not null"`
	RejectReason string
	Payload      RequestExpression `gorm:"type:json;not null"`
	CreatedAt    time.Time         `gorm:"not null"`
	UpdatedAt    time.Time
}

func (Table) TableName() string {
	return tableName
}

func (t *Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (t *Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}
