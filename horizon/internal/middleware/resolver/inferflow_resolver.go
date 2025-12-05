package resolver

const (
	screenTypeInferflowConfig         = "inferflow-config"
	screenTypeInferflowConfigApproval = "inferflow-config-approval"
	screenTypeInferflowConfigTesting  = "inferflow-config-testing"

	serviceInferflow = "inferflow"

	moduleReview     = "review"
	moduleReject     = "reject"
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
