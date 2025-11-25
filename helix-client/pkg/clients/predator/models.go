package predator

type PredatorRequest struct {
	ModelName    string   `json:"model_name"`
	ModelVersion string   `json:"model_version"`
	Inputs       []Input  `json:"inputs"`
	Outputs      []Output `json:"outputs"`
	// Request-level configuration (required)
	BatchSize int   `json:"batch_size"` // Batch size for this specific request
	Deadline  int64 `json:"deadline"`   // Request deadline in milliseconds
	// Sequence-related fields for stateful models
	SequenceId    *string `json:"sequence_id,omitempty"`    // Unique identifier for the sequence
	SequenceStart *bool   `json:"sequence_start,omitempty"` // Indicates if this is the start of a sequence
	SequenceEnd   *bool   `json:"sequence_end,omitempty"`   // Indicates if this is the end of a sequence
}

type Input struct {
	Name       string       `json:"name"`
	DataType   string       `json:"datatype"`
	Dims       []int        `json:"dims"`
	Data       [][][]byte   `json:"data"`
	StringData [][][]string `json:"string_data"`
}

type Output struct {
	Name        string   `json:"name"`
	ModelScores []string `json:"model_scores"`
	Dims        [][]int  `json:"dims"`
}

type PredatorResponse struct {
	ModelName    string           `json:"model_name"`
	ModelVersion string           `json:"model_version"`
	Outputs      []ResponseOutput `json:"outputs"`
}

type ResponseOutput struct {
	Name       string     `json:"name"`
	DataType   string     `json:"datatype"`
	Shape      []int64    `json:"shape"`
	Data       [][]byte   `json:"data"`
	StringData [][]string `json:"string_data"`
}
