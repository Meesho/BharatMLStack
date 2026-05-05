package circuitbreaker

import (
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupViperForManager(configs map[string]interface{}) {
	viper.Reset()
	for key, value := range configs {
		viper.Set(key, value)
	}
}

func TestNewManager(t *testing.T) {
	setupViperForManager(map[string]interface{}{
		"testService" + CBEnabled:                  true,
		"testService" + CBName:                     "testCB",
		"testService" + CBFailureRateThreshold:     50,
		"testService" + CBFailureRateMinimumWindow: 20,
		"testService" + CBFailureRateWindowInMs:    10000,
		"testService" + CBSuccessCountThreshold:    75,
		"testService" + CBSuccessCountWindow:       10,
		"testService" + CBWithDelayInMS:            1000,
		"testService" + CBVersion:                  1,
	})

	m := NewManager("testService")
	assert.NotNil(t, m)
	managerImpl, ok := m.(*manager)
	assert.True(t, ok)
	assert.NotNil(t, managerImpl.cbConfig)
	assert.True(t, managerImpl.cbConfig.Enabled)
	assert.Equal(t, "testService", managerImpl.envPrefix)
}

func TestGetOrCreateManualCB_CreateNew(t *testing.T) {
	setupViperForManager(map[string]interface{}{
		"testService" + CBEnabled:                  true,
		"testService" + CBName:                     "testCB",
		"testService" + CBFailureRateThreshold:     50,
		"testService" + CBFailureRateMinimumWindow: 20,
		"testService" + CBFailureRateWindowInMs:    10000,
		"testService" + CBSuccessCountThreshold:    75,
		"testService" + CBSuccessCountWindow:       10,
		"testService" + CBWithDelayInMS:            1000,
		"testService" + CBVersion:                  1,
	})
	m := NewManager("testService")
	cb, err := m.GetOrCreateManualCB("key1")
	assert.NoError(t, err)
	assert.NotNil(t, cb)
}

func TestGetOrCreateManualCB_ReturnExisting(t *testing.T) {
	setupViperForManager(map[string]interface{}{
		"testService" + CBEnabled:                  true,
		"testService" + CBName:                     "testCB",
		"testService" + CBFailureRateThreshold:     50,
		"testService" + CBFailureRateMinimumWindow: 20,
		"testService" + CBFailureRateWindowInMs:    10000,
		"testService" + CBSuccessCountThreshold:    75,
		"testService" + CBSuccessCountWindow:       10,
		"testService" + CBWithDelayInMS:            1000,
		"testService" + CBVersion:                  1,
	})
	m := NewManager("testService")
	cb1, err1 := m.GetOrCreateManualCB("key1")
	assert.NoError(t, err1)
	assert.NotNil(t, cb1)

	cb2, err2 := m.GetOrCreateManualCB("key1")
	assert.NoError(t, err2)
	assert.NotNil(t, cb2)

	assert.Same(t, cb1, cb2)
}

func TestGetOrCreateManualCB_Concurrent(t *testing.T) {
	setupViperForManager(map[string]interface{}{
		"testService" + CBEnabled:                  true,
		"testService" + CBName:                     "testCB",
		"testService" + CBFailureRateThreshold:     50,
		"testService" + CBFailureRateMinimumWindow: 20,
		"testService" + CBFailureRateWindowInMs:    10000,
		"testService" + CBSuccessCountThreshold:    75,
		"testService" + CBSuccessCountWindow:       10,
		"testService" + CBWithDelayInMS:            1000,
		"testService" + CBVersion:                  1,
	})
	m := NewManager("testService")
	var wg sync.WaitGroup
	numGoroutines := 100
	wg.Add(numGoroutines)

	cbs := make(chan ManualCircuitBreaker, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			cb, err := m.GetOrCreateManualCB("concurrent_key")
			assert.NoError(t, err)
			cbs <- cb
		}()
	}

	wg.Wait()
	close(cbs)

	var firstCb ManualCircuitBreaker
	isFirst := true
	for cb := range cbs {
		if isFirst {
			firstCb = cb
			isFirst = false
		}
		assert.NotNil(t, cb)
		assert.Same(t, firstCb, cb)
	}
}

func TestGetOrCreateManualCB_Disabled(t *testing.T) {
	setupViperForManager(map[string]interface{}{
		"testService" + CBEnabled: false,
	})
	m := NewManager("testService")
	cb, err := m.GetOrCreateManualCB("key1")
	assert.NoError(t, err)
	assert.NotNil(t, cb)
	assert.True(t, cb.IsAllowed(), "pass through breaker should always allow calls")
}
