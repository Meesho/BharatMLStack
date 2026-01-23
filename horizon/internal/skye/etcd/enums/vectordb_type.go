package enums

// VectorDbType represents the different types of models that can be used in the system.
type VectorDbType string

// Constants representing various model types.
const (
	QDRANT  VectorDbType = "QDRANT" // Indicates a model that resets the state.
	NGT     VectorDbType = "NGT"    // Indicates a model that applies changes incrementally.
	EIGENIX VectorDbType = "EIGENIX"
)
