package workflow

import "github.com/Meesho/BharatMLStack/skye/internal/config/enums"

type ModelStateExecutorPayload struct {
	Entity                string             `json:"entity"`
	Model                 string             `json:"model"`
	Variant               string             `json:"variant"`
	Version               int                `json:"version"`
	EmbeddingStoreVersion int                `json:"embedding_store_version"`
	VariantState          enums.VariantState `json:"variant_state"`
	Counter               int                `json:"counter"`
}
