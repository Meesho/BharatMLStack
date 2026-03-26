package resolver

const (
<<<<<<< HEAD:horizon/internal/middlewares/resolver/predator_resolver.go
	// Predator screen types
	screenTypePredatorDeployable = "deployable"
	screenTypePredatorModel      = "model"
	screenTypePredatorModelApproval = "model-approval"

	servicePredator = "predator"

	// Resolver function names (based on common patterns)
	resolverPredatorDeployableDiscovery = "PredatorDeployableDiscoveryResolver"
	resolverPredatorModelDiscovery       = "PredatorModelDiscoveryResolver"
	resolverPredatorModelRegistry        = "PredatorModelRegistryResolver"
	resolverPredatorModelEdit            = "PredatorModelEditResolver"
	resolverPredatorModelUpload          = "PredatorModelUploadResolver"
	resolverPredatorModelApproval        = "PredatorModelApprovalResolver"
	resolverPredatorModelApprovalView    = "PredatorModelApprovalViewResolver"
)

type predatorResolver struct {
}

func NewPredatorServiceResolver() (ServiceResolver, error) {
	return &predatorResolver{}, nil
}

func (r *predatorResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		// Deployable Discovery - View
		resolverPredatorDeployableDiscovery: StaticResolver(screenTypePredatorDeployable, moduleView, servicePredator),
		
		// Model Discovery - View
		resolverPredatorModelDiscovery: StaticResolver(screenTypePredatorModel, moduleView, servicePredator),
		
		// Model Registry - Onboard/Edit/Upload
		resolverPredatorModelRegistry: StaticResolver(screenTypePredatorModel, moduleOnboard, servicePredator),
		resolverPredatorModelEdit: StaticResolver(screenTypePredatorModel, moduleEdit, servicePredator),
		resolverPredatorModelUpload: StaticResolver(screenTypePredatorModel, "upload", servicePredator),
		
		// Model Approval - Approve/View
		resolverPredatorModelApproval: StaticResolver(screenTypePredatorModelApproval, moduleApprove, servicePredator),
		resolverPredatorModelApprovalView: StaticResolver(screenTypePredatorModelApproval, moduleView, servicePredator),
	}
}

=======
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
>>>>>>> 719e1f68b6c4710e883a4d61b281c16133c167a5:horizon/internal/middleware/resolver/predator_resolver.go
