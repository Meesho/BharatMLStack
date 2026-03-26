package resolver

const (
<<<<<<< HEAD:horizon/internal/middlewares/resolver/inferflow_resolver.go
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



=======
	screenTypeInferflowConfig         = "inferflow-config"
	screenTypeInferflowConfigApproval = "inferflow-config-approval"
	screenTypeInferflowConfigTesting  = "inferflow-config-testing"

	serviceInferflow = "inferflow"

	moduleReview     = "review"
	moduleReject     = "reject"
	moduleApprove    = "approve"
	moduleDeactivate = "deactivate"
	moduleClone      = "clone"
	moduleTest       = "test"

	moduleEdit                           = "edit"
	moduleCancel                         = "cancel"
	moduleInferflowScaleUp               = "scale-up"
	resolverInferflowRequestReview       = "InferflowRequestReviewResolver"
	resolverInferflowRequestCancel       = "InferflowRequestCancelResolver"
	resolverInferflowRequestValidate     = "InferflowRequestValidateResolver"
	resolverInferflowRequestDiscovery    = "InferflowRequestDiscoveryResolver"
	resolverInferflowDiscovery           = "InferflowDiscoveryResolver"
	resolverInferflowScaleUp             = "InferflowScaleUpResolver"
	resolverInferflowDelete              = "InferflowDeleteResolver"
	resolverInferflowClone               = "InferflowCloneResolver"
	resolverInferflowPromote             = "InferflowPromoteResolver"
	resolverInferflowOnboard             = "InferflowOnboardResolver"
	resolverInferflowEdit                = "InferflowEditResolver"
	resolverInferflowTestGenerateRequest = "InferflowTestGenerateRequestResolver"
	resolverInferflowTestExecuteRequest  = "InferflowTestExecuteRequestResolver"
)

type InferflowResolver struct {
}

func NewInferflowServiceResolver() (ServiceResolver, error) {
	return &InferflowResolver{}, nil
}

func (r *InferflowResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		resolverInferflowRequestReview:       StaticResolver(screenTypeInferflowConfigApproval, moduleReview, serviceInferflow),
		resolverInferflowRequestCancel:       StaticResolver(screenTypeInferflowConfigApproval, moduleCancel, serviceInferflow),
		resolverInferflowRequestValidate:     StaticResolver(screenTypeInferflowConfigApproval, moduleValidate, serviceInferflow),
		resolverInferflowRequestDiscovery:    StaticResolver(screenTypeInferflowConfigApproval, moduleView, serviceInferflow),
		resolverInferflowDiscovery:           StaticResolver(screenTypeInferflowConfig, moduleView, serviceInferflow),
		resolverInferflowScaleUp:             StaticResolver(screenTypeInferflowConfig, moduleInferflowScaleUp, serviceInferflow),
		resolverInferflowDelete:              StaticResolver(screenTypeInferflowConfig, moduleDelete, serviceInferflow),
		resolverInferflowClone:               StaticResolver(screenTypeInferflowConfig, moduleClone, serviceInferflow),
		resolverInferflowPromote:             StaticResolver(screenTypeInferflowConfig, modulePromote, serviceInferflow),
		resolverInferflowOnboard:             StaticResolver(screenTypeInferflowConfig, moduleOnboard, serviceInferflow),
		resolverInferflowEdit:                StaticResolver(screenTypeInferflowConfig, moduleEdit, serviceInferflow),
		resolverInferflowTestGenerateRequest: StaticResolver(screenTypeInferflowConfigTesting, moduleTest, serviceInferflow),
		resolverInferflowTestExecuteRequest:  StaticResolver(screenTypeInferflowConfigTesting, moduleTest, serviceInferflow),
	}
}
>>>>>>> 719e1f68b6c4710e883a4d61b281c16133c167a5:horizon/internal/middleware/resolver/inferflow_resolver.go
