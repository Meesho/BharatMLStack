package circuitbreaker

import (
	"fmt"
	"sync"
)

type Manager interface {
	GetOrCreateManualCB(key string) (ManualCircuitBreaker, error)
}

type manager struct {
	breakers  sync.Map
	cbConfig  *Config
	envPrefix string
}

func NewManager(envPrefix string) Manager {
	return &manager{
		breakers:  sync.Map{},
		envPrefix: envPrefix,
		cbConfig:  BuildConfig(envPrefix),
	}
}

func (m *manager) GetOrCreateManualCB(key string) (ManualCircuitBreaker, error) {
	if m.cbConfig == nil {
		return nil, fmt.Errorf("circuit breaker config is nil")
	}

	if breaker, ok := m.breakers.Load(key); ok {
		if typedBreaker, castOk := breaker.(ManualCircuitBreaker); castOk {
			return typedBreaker, nil
		}
	}

	newBreaker := GetManualCircuitBreaker(m.cbConfig)
	actual, _ := m.breakers.LoadOrStore(key, newBreaker)
	if typedBreaker, castOk := actual.(ManualCircuitBreaker); castOk {
		return typedBreaker, nil
	}

	return nil, fmt.Errorf("item in sync.Map is not a ManualCircuitBreaker")
}
