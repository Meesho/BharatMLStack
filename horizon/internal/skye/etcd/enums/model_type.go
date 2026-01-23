package enums

// ModelType represents the different types of models that can be used in the system.
type ModelType string

// Constants representing various model types.
const (
	RESET ModelType = "RESET" // Indicates a model that resets the state.
	DELTA ModelType = "DELTA" // Indicates a model that applies changes incrementally.
)
