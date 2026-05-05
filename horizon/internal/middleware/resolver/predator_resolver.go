package resolver

const (
	screenTypeApproval = "model-approval"
	screenTypeModel    = "model"
	servicePredator    = "predator"
	moduleView         = "view"
	moduleDelete       = "delete"
	moduleScaleUp      = "scale_up"
	modulePromote      = "promote"
	moduleOnboard      = "onboard"
	moduleValidate     = "validate"

	// Resolvers
	resolverModelRequestApprove      = "ModelRequestApproveResolver"
	resolverModelRequestReject       = "ModelRequestRejectResolver"
	resolverModelDiscovery           = "ModelDiscoveryResolver"
	resolverModelDelete              = "ModelDeleteResolver"
	resolverModelScaleUp             = "ModelScaleUpResolver"
	resolverModelPromote             = "ModelPromoteResolver"
	resolverModelOnboard             = "ModelOnboardResolver"
	resolverModelParams              = "ModelParamsResolver"
	resolverModelRequestDiscovery    = "ModelRequestDiscoveryResolver"
	resolverModelValidator           = "ModelValidatorResolver"
	resolverModelUploadMetaData      = "ModelUploadMetaDataResolver"
	resolverModelSourceDiscovery     = "ModelSourceDiscoveryResolver"
	resolverModelTestGenerateRequest = "ModelTestGenerateRequestResolver"
	resolverModelFunctionalTest      = "ModelFunctionalTestResolver"
	resolverModelLoadTest            = "ModelLoadTestResolver"
)

type PredatorResolver struct{}

func NewPredatorServiceResolver() (ServiceResolver, error) {
	return &PredatorResolver{}, nil
}

func (p *PredatorResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		resolverModelRequestApprove:      StaticResolver(screenTypeApproval, moduleApprove, servicePredator),
		resolverModelRequestReject:       StaticResolver(screenTypeApproval, moduleReject, servicePredator),
		resolverModelDiscovery:           StaticResolver(screenTypeModel, moduleView, servicePredator),
		resolverModelDelete:              StaticResolver(screenTypeModel, moduleDelete, servicePredator),
		resolverModelScaleUp:             StaticResolver(screenTypeModel, moduleScaleUp, servicePredator),
		resolverModelPromote:             StaticResolver(screenTypeModel, modulePromote, servicePredator),
		resolverModelOnboard:             StaticResolver(screenTypeModel, moduleOnboard, servicePredator),
		resolverModelParams:              StaticResolver(screenTypeModel, moduleOnboard, servicePredator),
		resolverModelRequestDiscovery:    StaticResolver(screenTypeApproval, moduleView, servicePredator),
		resolverModelValidator:           StaticResolver(screenTypeApproval, moduleValidate, servicePredator),
		resolverModelUploadMetaData:      StaticResolver(screenTypeApproval, moduleReview, servicePredator),
		resolverModelSourceDiscovery:     StaticResolver(screenTypeModel, moduleOnboard, servicePredator),
		resolverModelTestGenerateRequest: StaticResolver(screenTypeModel, moduleTest, servicePredator),
		resolverModelFunctionalTest:      StaticResolver(screenTypeModel, moduleTest, servicePredator),
		resolverModelLoadTest:            StaticResolver(screenTypeModel, moduleTest, servicePredator),
	}
}
