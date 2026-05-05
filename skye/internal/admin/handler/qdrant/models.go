package qdrant

import "github.com/Meesho/BharatMLStack/skye/internal/config/enums"

type CreateCollectionRequest struct {
	Entity  string `json:"entity"`
	Model   string `json:"model"`
	Variant string `json:"variant"`
	Version int    `json:"version"`
}

type TriggerIndexingRequest struct {
	Entity                string         `json:"entity"`
	Model                 string         `json:"model"`
	VariantsVersionMap    map[string]int `json:"variants_version_map"`
	EmbeddingStoreVersion int            `json:"embedding_store_version"`
	Partition             string         `json:"partition"`
}

type ProcessModelRequest struct {
	Entity       string             `json:"entity"`
	Model        string             `json:"model"`
	VectorDbType enums.VectorDbType `json:"vector_db_type"`
}

type ProcessMultiVariantRequest struct {
	Entity       string             `json:"entity"`
	Model        string             `json:"model"`
	Variants     []string           `json:"variants"`
	VectorDbType enums.VectorDbType `json:"vector_db_type"`
}

type ProcessModelsWithFrequencyRequest struct {
	Entity       string             `json:"entity"`
	Frequency    string             `json:"frequency"`
	VectorDbType enums.VectorDbType `json:"vector_db_type"`
}

type ProcessModelsWithFrequencyResponse struct {
	ProcessModelResponse []ProcessFreqModelResponse `json:"response"`
}

type ProcessModelResponse struct {
	Variants              map[string]int  `json:"variants"`
	KafkaId               int             `json:"kafka_id"`
	TrainingDataPath      string          `json:"training_data_path"`
	EmbeddingStoreVersion int             `json:"embedding_store_version"`
	Model                 string          `json:"model"`
	NumberOfPartitions    int             `json:"number_of_partitions"`
	TopicName             string          `json:"topic_name"`
	ModelType             enums.ModelType `json:"model_type"`
}

type ProcessFreqModelResponse struct {
	Model string `json:"model"`
}

type DeleteCollectionRequest struct {
	Entity  string `json:"entity"`
	Model   string `json:"model"`
	Variant string `json:"variant"`
	Version int    `json:"version"`
}

type GetCollectionInfoRequest struct {
	Entity      string     `json:"entity"`
	Model       string     `json:"model"`
	Variant     string     `json:"variant"`
	VariantType enums.Type `json:"variant_type"`
}

type PromoteVariantRequest struct {
	Entity       string             `json:"entity"`
	Model        string             `json:"model"`
	Variant      string             `json:"variant"`
	VectorDbType enums.VectorDbType `json:"vector_db_type"`
	Host         string             `json:"host"`
}
