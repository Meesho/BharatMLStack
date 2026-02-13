package registry

type Manager interface {
	RegisterStore(*CreateStoreRequest) error
	RegisterFrequency(request *CreateFrequencyRequest) error
	RegisterEntity(*RegisterEntityRequest) error
	RegisterModel(*RegisterModelRequest) error
	RegisterVariant(*RegisterVariantRequest) error
}
