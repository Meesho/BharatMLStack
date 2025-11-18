package config

var InferflowConfigStruct InferflowConfigStruct

type InMemoryCache struct {
	SizeInBytes int `mapstructure:"size-in-bytes"`
}

type InferflowConfigStruct struct {
	InMemoryCache InMemoryCache `mapstructure:"in-memory-cache"`
}

func GetInferflowConfigInstance() *InferflowConfigStruct {
	return &InferflowConfigStruct
}
