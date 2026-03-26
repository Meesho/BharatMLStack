package numerix_config

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"

	"gorm.io/gorm"
)

const (
	tableName = "numerix_config"
)

type Expression struct {
	InfixExpression   string `json:"infix_expression"`
	PostfixExpression string `json:"postfix_expression"`
}

func (r *Expression) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for Expression")
	}
	return json.Unmarshal(bytes, r)
}

func (r Expression) Value() (driver.Value, error) {
	return json.Marshal(r)
}

type Table struct {
	ID          uint       `gorm:"primaryKey;autoIncrement"`
	ConfigID    uint       `gorm:"unique"`
	ConfigValue Expression `gorm:"type:json;not null"`
	CreatedBy   string     `gorm:"not null"`
	UpdatedBy   string
	Active      bool      `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time
	TestResults json.RawMessage
}

func (Table) TableName() string {
	return tableName
}

func (Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
