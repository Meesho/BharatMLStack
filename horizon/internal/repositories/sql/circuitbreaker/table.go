package circuitbreaker

import (
	"gorm.io/gorm"
	"time"
)

const cbTableName = "circuit_breakers"
const createdAt = "CreatedAt"
const updatedAt = "UpdatedAt"

type CircuitBreaker struct {
	ID                       int    `gorm:"primaryKey"`
	Name                     string `gorm:"unique"`
	Enabled                  bool   `gorm:"default:true"`
	SlowCallRateThreshold    int
	FailureRateThreshold     int
	SlidingWindowType        string `gorm:"not null"`
	SlidingWindowSize        int    `gorm:"not null"`
	MinimumNoOfCalls         int
	AutoTransitionOpenToHalf bool `gorm:"default:true"`
	MaxWaitHalfOpenState     int
	WaitOpenState            int
	SlowCallDuration         int
	PermittedCallsHalfOpen   int
	BHName                   int `gorm:"unique"`
	BHCorePoolSize           int
	BHMaxPoolSize            int
	BHQueueSize              int
	BHFallbackCoreSize       int
	BHKeepAliveMs            int
	CreatedBy                string
	UpdatedBy                string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (CircuitBreaker) TableName() string {
	return cbTableName
}

func (CircuitBreaker) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (CircuitBreaker) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}
