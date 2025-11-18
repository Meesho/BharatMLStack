package connectionconfig

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"

	"gorm.io/gorm"
)

const ConnectionConfigTableName = "service_connection_config"

type GrpcConfig struct {
	Deadline              int    `json:"deadline"`
	PlainText             bool   `json:"plain_text"`
	GrpcChannelAlgorithm  string `json:"grpc_channel_algorithm"`
	ChannelThreadPoolSize int    `json:"channel_thread_pool_size"`
	BoundedQueueSize      int    `json:"bounded_queue_size"`
}

// Value implements the driver.Valuer interface for GrpcConfig
func (g GrpcConfig) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// Scan implements the sql.Scanner interface for GrpcConfig
func (g *GrpcConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &g)
}

type HttpConfig struct {
	Timeout               int `json:"timeout"`
	MaxIdleConnection     int `json:"max_idle_connection"`
	MaxConnectionPerHost  int `json:"max_connection_per_host"`
	IdleConnectionTimeout int `json:"idle_connection_timeout"`
	KeepAliveTime         int `json:"keep_alive_time"`
}

// Value implements the driver.Valuer interface for HttpConfig
func (h HttpConfig) Value() (driver.Value, error) {
	return json.Marshal(h)
}

// Scan implements the sql.Scanner interface for HttpConfig
func (h *HttpConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &h)
}

type ConnectionConfig struct {
	Id           uint       `gorm:"primaryKey;autoIncrement"`
	Default      bool       `gorm:"not null"`
	Service      string     `gorm:"not null"`
	ConnProtocol string     `gorm:"not null"`
	HttpConfig   HttpConfig `gorm:"type:json;not null"`
	GrpcConfig   GrpcConfig `gorm:"type:json;not null"`
	Active       bool       `gorm:"not null"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time
	CreatedBy    string `gorm:"not null"`
	UpdatedBy    string
}

func (ConnectionConfig) TableName() string {
	return ConnectionConfigTableName
}

func (ConnectionConfig) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.CreatedAt, time.Now())
	return
}

func (ConnectionConfig) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(constant.UpdatedAt, time.Now())
	return
}
