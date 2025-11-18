package external

const (
	ERROR_TYPE = "error-type"
	IS_API_ERR = "interaction-store-api-error"
)

type ISConfig struct {
	Host       string `koanf:"isHost"`
	Port       string `koanf:"isPort"`
	DeadLine   int    `koanf:"isdeadLine"`
	PLAIN_TEXT bool   `koanf:"isGrpcPlainText"`
}

type ISResponse struct {
	InteractionTypeToInteractions []InteractionTypeToInteraction
}

type InteractionTypeToInteraction struct {
	Interactions    []Interaction
	InteractionType string
}

type Interaction struct {
	Id        string
	Type      string
	TimeStamp string
}
