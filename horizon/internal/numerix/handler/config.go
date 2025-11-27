package handler

type Config interface {
	Onboard(OnboardConfigRequest) (Response, error)
	Promote(PromoteConfigRequest) (Response, error)
	GetAll(GetAllConfigsRequest) (GetAllConfigsResponse, error)
	GenerateExpression(ExpressionGenerateRequest) (ExpressionGenerateResponse, error)
	GetExpressionVariables(ExpressionVariablesRequest) (ExpressionVariablesResponse, error)
	ReviewRequest(ReviewRequestConfigRequest) (Response, error)
	Edit(EditConfigRequest) (Response, error)
	CancelRequest(CancelConfigRequest) (Response, error)
	GetAllRequests(GetAllRequestConfigsRequest) (GetAllRequestConfigsResponse, error)
	GenerateFuncitonalTestRequest(RequestGenerationRequest) (FuncitonalRequestGenerationResponse, error)
	ExecuteFuncitonalTestRequest(ExecuteRequestFunctionalRequest) (ExecuteRequestFunctionalResponse, error)
}
