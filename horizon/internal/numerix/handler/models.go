package handler

import (
	"encoding/json"
	"time"
)

type ExpressionGenerateRequest struct {
	Expression string `json:"infix_expression"`
}

type ExpressionGenerateResponse struct {
	Error string `json:"error"`
	Data  string `json:"data"`
}

type ExpressionVariablesRequest struct {
	ConfigID uint `uri:"config_id"`
}

type ExpressionVariablesResponse struct {
	Error string   `json:"error"`
	Data  []string `json:"data"`
}

type GetAllConfigsResponse struct {
	Error string          `json:"error"`
	Data  []NumerixConfig `json:"data"`
}

type NumerixConfig struct {
	ConfigID                uint            `json:"config_id"`
	InfixExpression         string          `json:"infix_expression"`
	PostfixExpression       string          `json:"postfix_expression"`
	CreatedBy               string          `json:"created_by"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedBy               string          `json:"updated_by"`
	UpdatedAt               time.Time       `json:"updated_at"`
	DeployableRunningStatus bool            `json:"deployable_running_status"`
	MonitoringUrl           string          `json:"monitoring_url"`
	TestResults             json.RawMessage `json:"test_results"`
}

type Expression struct {
	InfixExpression   string `json:"infix_expression"`
	PostfixExpression string `json:"postfix_expression"`
}

type OnboardConfigRequest struct {
	Payload struct {
		ConfigValue Expression `json:"config_value"`
	} `json:"payload"`
	CreatedBy string `json:"created_by"`
}

type EditConfigRequest struct {
	Payload struct {
		ConfigID    uint       `json:"config_id"`
		ConfigValue Expression `json:"config_value"`
	} `json:"payload"`
	CreatedBy string `json:"created_by"`
}

type CancelConfigRequest struct {
	RequestID uint   `json:"request_id"`
	UpdatedBy string `json:"updated_by"`
}

type GetAllRequestConfigsRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type GetAllRequestConfigsResponse struct {
	Error string          `json:"error"`
	Data  []RequestConfig `json:"data"`
}

type RequestExpression struct {
	ConfigValue Expression `json:"config_value"`
}

type RequestConfig struct {
	RequestID    uint              `json:"request_id"`
	ComputeId    uint              `json:"compute_id"`
	Payload      RequestExpression `json:"payload"`
	CreatedBy    string            `json:"created_by"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedBy    string            `json:"updated_by"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Status       string            `json:"status"`
	RequestType  string            `json:"request_type"`
	RejectReason string            `json:"reject_reason"`
	Reviewer     string            `json:"reviewer"`
}

type ReviewRequestConfigRequest struct {
	RequestID    uint   `json:"request_id"`
	Status       string `json:"status"`
	Reviewer     string `json:"reviewer"`
	RejectReason string `json:"reject_reason"`
}

type PromoteConfigRequest struct {
	Payload struct {
		ConfigID    uint       `json:"config_id"`
		ConfigValue Expression `json:"config_value"`
	} `json:"payload"`
	UpdatedBy string `json:"updated_by"`
}

type Message struct {
	Message string `json:"message"`
}

type Response struct {
	Error string  `json:"error"`
	Data  Message `json:"data"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type RequestGenerationRequest struct {
	ComputeId string `json:"compute_id"`
	BatchSize string `json:"batch_size"`
	DataType  string `json:"data_type"`
}

type FuncitonalRequestGenerationResponse struct {
	ComputeId   string      `json:"compute_id"`
	DataType    string      `json:"data_type"`
	RequestBody RequestBody `json:"request_body"`
}

type RequestBody struct {
	EntityScoreData EntityScoreData `json:"entity_score_data"`
}

type EntityScoreData struct {
	Schema       []string `json:"schema"`
	EntityScores []Data   `json:"entity_scores"`
}

type Data struct {
	StringData DataValues `json:"string_data"`
}

type DataValues struct {
	Values []string `json:"values"`
}
type ExecuteRequestFunctionalRequest struct {
	DataType    string      `json:"data_type"`
	EndPoint    string      `json:"end_point"`
	ComputeId   string      `json:"compute_id"`
	RequestBody RequestBody `json:"request_body"`
}

type ExecuteRequestFunctionalResponse struct {
	ComputationalScoreData ComputationalScoreData `json:"computational_score_data"`
}

type ComputationalScoreData struct {
	Schema              []string             `json:"schema"`
	ComputationalScores []ComputationalScore `json:"computational_scores"`
}

type ComputationalScore struct {
	Data Data `json:"data"`
}

type BinaryOp struct {
	Operator   string `json:"operator"`
	Precedence uint   `json:"precedence"`
}

type UnaryOp struct {
	Operator   string `json:"operator"`
	Parameters uint   `json:"parameters"`
}

type GetBinaryOpsResponse struct {
	Error string     `json:"error"`
	Data  []BinaryOp `json:"data"`
}

type GetUnaryOpsResponse struct {
	Error string    `json:"error"`
	Data  []UnaryOp `json:"data"`
}
