package handler

type Config interface {
	Onboard(OnboardRequest) (Response, error)
	GetAll() (GetAllResponse, error)
	Edit(EditRequest) (Response, error)
}
