package handler

type OnboardRequestPayload struct {
	Bu      string `json:"bu"`
	Team    string `json:"team"`
	Service string `json:"service"`
}

type OnboardRequest struct {
	Payload   OnboardRequestPayload `json:"payload"`
	CreatedBy string                `json:"created_by"`
}

type Message struct {
	Message string `json:"message"`
}

type Response struct {
	Error string  `json:"error"`
	Data  Message `json:"data"`
}

type ApplicationConfig struct {
	AppToken  string `json:"app_token"`
	Active    bool   `json:"active"`
	Bu        string `json:"bu"`
	Team      string `json:"team"`
	Service   string `json:"service"`
	CreatedBy string `json:"created_by"`
	UpdatedBy string `json:"updated_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type GetAllResponse struct {
	Data []ApplicationConfig `json:"data"`
}

type EditRequestPayload struct {
	AppToken  string `json:"app_token"`
	Bu        string `json:"bu"`
	Team      string `json:"team"`
	Service   string `json:"service"`
	UpdatedBy string `json:"updated_by"`
}

type EditRequest struct {
	Payload   EditRequestPayload `json:"payload"`
	CreatedBy string             `json:"created_by"`
}
