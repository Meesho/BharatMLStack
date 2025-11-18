package featurestore

const (
	configPrefix        = "externalServiceOnFs"
	errorType           = "error-type"
	onFsApiReqTimeOut   = "featurestore-api-request-timeout"
	onFsApiReqCancelled = "featurestore-api-request-cancelled"
	onFsApiError        = "featurestore-api-error"
)

type FSConfig struct {
	Host        string `koanf:"fsHost"`
	Port        string `koanf:"fsPort"`
	CallerId    string `koanf:"fsCallerId"`
	CallerToken string `koanf:"fsCallerToken"`
	DeadLine    int    `koanf:"fsdeadLine"`
	BatchSize   int    `koanf:"fsBatchSize"`
	PLAIN_TEXT  bool   `koanf:"fsGrpcPlainText"`
}
