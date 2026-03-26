package resolver

const (
	// InferFlow screen types
	screenTypeInferFlowDeployable      = "deployable"
	screenTypeInferFlowConnectionConfig = "connection-config"
	screenTypeInferFlowMPConfig        = "mp-config"
	screenTypeInferFlowMPConfigApproval = "mp-config-approval"

	serviceInferFlow = "inferflow"

	// Resolver function names (based on common patterns)
	resolverInferFlowDeployableDiscovery     = "InferFlowDeployableDiscoveryResolver"
	resolverInferFlowConnectionConfig        = "InferFlowConnectionConfigResolver"
	resolverInferFlowMPConfigRegistry       = "InferFlowMPConfigRegistryResolver"
	resolverInferFlowMPConfigEdit            = "InferFlowMPConfigEditResolver"
	resolverInferFlowMPConfigDiscovery      = "InferFlowMPConfigDiscoveryResolver"
	resolverInferFlowMPConfigApproval        = "InferFlowMPConfigApprovalResolver"
	resolverInferFlowMPConfigApprovalView    = "InferFlowMPConfigApprovalViewResolver"
)

type inferflowResolver struct {
}

func NewInferFlowServiceResolver() (ServiceResolver, error) {
	return &inferflowResolver{}, nil
}

func (r *inferflowResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		// Deployable Discovery - View
		resolverInferFlowDeployableDiscovery: StaticResolver(screenTypeInferFlowDeployable, moduleView, serviceInferFlow),
		
		// Connection Config - View/Edit
		resolverInferFlowConnectionConfig: StaticResolver(screenTypeInferFlowConnectionConfig, moduleView, serviceInferFlow),
		
		// MP Config Registry - Onboard/Edit
		resolverInferFlowMPConfigRegistry: StaticResolver(screenTypeInferFlowMPConfig, moduleOnboard, serviceInferFlow),
		resolverInferFlowMPConfigEdit: StaticResolver(screenTypeInferFlowMPConfig, moduleEdit, serviceInferFlow),
		
		// MP Config Discovery - View
		resolverInferFlowMPConfigDiscovery: StaticResolver(screenTypeInferFlowMPConfig, moduleView, serviceInferFlow),
		
		// MP Config Approval - Approve/View
		resolverInferFlowMPConfigApproval: StaticResolver(screenTypeInferFlowMPConfigApproval, moduleApprove, serviceInferFlow),
		resolverInferFlowMPConfigApprovalView: StaticResolver(screenTypeInferFlowMPConfigApproval, moduleView, serviceInferFlow),
	}
}



