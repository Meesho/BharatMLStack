package enums

// ModelType represents the different types of models that can be used in the system.
type FilterCondition string

// Constants representing various model types.
const (
	EQUALS     ModelType = "EQUALS"     // Indicates a model that resets the state.
	NOT_EQUALS ModelType = "NOT_EQUALS" // Indicates a model that applies changes incrementally.
)
