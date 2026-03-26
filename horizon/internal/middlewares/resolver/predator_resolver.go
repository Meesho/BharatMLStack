package resolver

const (
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

