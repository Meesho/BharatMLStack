package inferflow

import pb "github.com/Meesho/BharatMLStack/inferflow/server/grpc"

type InferflowRequest struct {
	Entity        *[]string
	EntityIds     *[][]string
	Features      *map[string][]string
	ModelConfigId string
	TrackingId    string
}

type InferflowResponse struct {
	ComponentData [][]string `json:"component_data"`
	Error         *Error     `json:"error"`
}

type Error struct {
	Message string `json:"message"`
}

type Inferflow struct {
	pb.InferflowServer
}
