package externalcall

import "sync"

var (
	// Working environment - generalized variable (not RingMaster specific)
	workingEnvironment string
	initWorkingEnvOnce sync.Once
)

// InitWorkingEnvironment initializes the working environment from config
func InitWorkingEnvironment(workingEnv string) {
	initWorkingEnvOnce.Do(func() {
		workingEnvironment = workingEnv
	})
}

// GetWorkingEnvironment returns the working environment (generalized, not RingMaster specific)
func GetWorkingEnvironment() string {
	return workingEnvironment
}

// GetRingmasterEnvironment returns the working environment (deprecated - use GetWorkingEnvironment instead)
// Kept for backward compatibility
func GetRingmasterEnvironment() string {
	return GetWorkingEnvironment()
}
