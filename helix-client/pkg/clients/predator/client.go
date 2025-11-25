package predator

type Client interface {
	GetInferenceScore(req *PredatorRequest) (*PredatorResponse, error)
}

