package handler

type Config interface {
	Onboard(request InferflowOnboardRequest, token string) (Response, error)
	Review(request ReviewRequest) (Response, error)
	Promote(request PromoteConfigRequest) (Response, error)
	Edit(request EditConfigOrCloneConfigRequest, token string) (Response, error)
	Clone(request EditConfigOrCloneConfigRequest, token string) (Response, error)
	Delete(request DeleteConfigRequest) (Response, error)
	ScaleUp(request ScaleUpConfigRequest) (Response, error)
	Cancel(request CancelConfigRequest) (Response, error)
	GetAll() (GetAllResponse, error)
	GetAllRequests(request GetAllRequestConfigsRequest) (GetAllRequestConfigsResponse, error)
	ValidateRequest(request ValidateRequest, token string) (Response, error)
	GenerateFunctionalTestRequest(request GenerateRequestFunctionalTestingRequest) (GenerateRequestFunctionalTestingResponse, error)
	ExecuteFuncitonalTestRequest(request ExecuteRequestFunctionalTestingRequest) (ExecuteRequestFunctionalTestingResponse, error)
	GetLatestRequest(requestID string) (GetLatestRequestResponse, error)
	GetLoggingTTL() (GetLoggingTTLResponse, error)
}
