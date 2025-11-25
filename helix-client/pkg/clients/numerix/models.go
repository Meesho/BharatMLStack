package numerix

import "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/numerix/client/grpc"

type NumerixRequest struct {
	EntityScoreData EntityScoreData `json:"entity_score_data"`
}

type EntityScoreData struct {
	Schema     []string   `json:"schema"`
	Data       [][][]byte `json:"data"`
	StringData [][]string `json:"string_data"`
	ComputeID  string     `json:"compute_id"`
	DataType   string     `json:"data_type"`
}

type NumerixResponse struct {
	ComputationScoreData ComputationScoreData `json:"computation_score_data"`
}

type ComputationScoreData struct {
	Schema     []string   `json:"schema"`
	Data       [][][]byte `json:"data"`
	StringData [][]string `json:"string_data"`
}

type NumerixRequestWrapper struct {
	RequestProto *grpc.NumerixRequestProto
}
