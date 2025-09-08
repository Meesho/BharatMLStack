package circuitbreaker

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type ManagerFactory struct {
	managers sync.Map
}

var (
	factory *ManagerFactory
	once    sync.Once
)

type Manager interface {
	GetOrCreateManualCB(key string) (ManualCircuitBreaker, error)
	ActivateCBKey([]string)
	DeactivateCBKey([]string)
	UpdateCBConfig(Config) error
	GetCBConfig() Config
	IsCBEnabled(key string) bool
	ForceOpenCB(key string) error
	ForceCloseCB(key string) error
	NormalExecutionModeCB(key string) error
}

type manager struct {
	breakers  sync.Map
	cbEnabled sync.Map
	cbConfig  *Config
	envPrefix string
}

func GetFactory() *ManagerFactory {
	once.Do(func() {
		factory = &ManagerFactory{
			managers: sync.Map{},
		}
	})
	return factory
}

func GetManager(envPrefix string) Manager {
	factory := GetFactory()
	manager, _ := factory.managers.LoadOrStore(envPrefix, &manager{
		breakers:  sync.Map{},
		cbEnabled: sync.Map{},
		envPrefix: envPrefix,
		cbConfig:  BuildConfig(envPrefix),
	})
	return manager.(Manager)
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

func (m *manager) ActivateCBKey(activeCBKeys []string) {
	for _, cbKey := range activeCBKeys {
		if _, ok := m.breakers.Load(cbKey); !ok {
			m.breakers.Store(cbKey, GetManualCircuitBreaker(m.cbConfig))
		}
		m.cbEnabled.Store(cbKey, true)
	}
}

func (m *manager) GetCBConfig() Config {
	return *m.cbConfig
}

func (m *manager) DeactivateCBKey(inactiveCBKeys []string) {
	for _, key := range inactiveCBKeys {
		m.breakers.Delete(key)
		m.cbEnabled.Delete(key)
	}
}

func (m *manager) UpdateCBConfig(cbConfig Config) error {
	m.cbConfig = &cbConfig
	m.breakers.Range(func(key, value interface{}) bool {
		if _, ok := value.(ManualCircuitBreaker); ok {
			newBreaker := GetManualCircuitBreaker(&cbConfig)
			m.breakers.Store(key, newBreaker)
		}
		return true
	})
	return nil
}

func (m *manager) IsCBEnabled(key string) bool {
	if value, ok := m.cbEnabled.Load(key); ok {
		if enabled, castOk := value.(bool); castOk {
			return enabled
		}
	}
	log.Debug().Msgf("No value found for key %s, returning false", key)
	return false
}

// ForceOpenCB brings the circuit breaker to force open state
func (m *manager) ForceOpenCB(key string) {
	breaker, err := m.GetOrCreateManualCB(key)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get circuit breaker for key %s", key)
		return
	}
	breaker.ForceOpen()
}

// ForceCloseCB brings the circuit breaker to force close state
func (m *manager) ForceCloseCB(key string) {
	breaker, err := m.GetOrCreateManualCB(key)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get circuit breaker for key %s", key)
		return
	}
	breaker.ForceClose()
}

// NormalExecutionModeCB brings the circuit breaker to normal execution mode
func (m *manager) NormalExecutionModeCB(key string) {
	breaker, err := m.GetOrCreateManualCB(key)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get circuit breaker for key %s", key)
		return
	}
	breaker.NormalExecutionMode()
}
