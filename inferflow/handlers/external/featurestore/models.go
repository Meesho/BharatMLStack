package featurestore

const (
	configPrefix        = "externalServiceOnFs"
	errorType           = "error-type"
	onFsApiReqTimeOut   = "featurestore-api-request-timeout"
	onFsApiReqCancelled = "featurestore-api-request-cancelled"
	onFsApiError        = "featurestore-api-error"
)

type FSConfig struct {
	Host        string `json:"fsHost"`
	Port        string `json:"fsPort"`
	CallerId    string `json:"fsCallerId"`
	CallerToken string `json:"fsCallerToken"`
	DeadLine    int    `json:"fsdeadLine"`
	BatchSize   int    `json:"fsBatchSize"`
	PLAIN_TEXT  bool   `json:"fsGrpcPlainText"`
}
