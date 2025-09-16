package resolver

const (
	screenTypeNumerixConfig         = "numerix-config"
	screenTypeNumerixConfigApproval = "numerix-config-approval"
	screenTypeNumerixConfigTesting  = "numerix-config-testing"

	serviceNumerix = "numerix"

	moduleView    = "view"
	moduleOnboard = "onboard"
	modulePromote = "promote"
	moduleEdit    = "edit"
	moduleReview  = "review"
	moduleCancel  = "cancel"
	moduleDelete  = "delete"
	moduleTest    = "test"

	resolverNumerixExpressionGenerate     = "NumerixExpressionGenerateResolver"
	resolverNumerixExpressionVariables    = "NumerixExpressionVariablesResolver"
	resolverNumerixConfigOnboard          = "NumerixConfigOnboardResolver"
	resolverNumerixConfigPromote          = "NumerixConfigPromoteResolver"
	resolverNumerixConfigDiscovery        = "NumerixConfigDiscoveryResolver"
	resolverNumerixConfigRequestReview    = "NumerixConfigRequestReviewResolver"
	resolverNumerixConfigRequestCancel    = "NumerixConfigRequestCancelResolver"
	resolverNumerixConfigEdit             = "NumerixConfigEditResolver"
	resolverNumerixConfigRequestDiscovery = "NumerixConfigRequestDiscoveryResolver"
	resolverNumerixConfigRequestDelete    = "NumerixConfigRequestDeleteResolver"
	resolverNumerixTestGenerateRequest    = "NumerixTestGenerateRequestResolver"
	resolverNumerixTestExecuteRequest     = "NumerixTestExecuteRequestResolver"
)

type numerixResolver struct {
}

func NewnumerixServiceResolver() (ServiceResolver, error) {
	return &numerixResolver{}, nil
}

func (r *numerixResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		resolverNumerixExpressionGenerate:     StaticResolver(screenTypeNumerixConfig, moduleView, serviceNumerix),
		resolverNumerixExpressionVariables:    StaticResolver(screenTypeNumerixConfig, moduleView, serviceNumerix),
		resolverNumerixConfigOnboard:          StaticResolver(screenTypeNumerixConfig, moduleOnboard, serviceNumerix),
		resolverNumerixConfigPromote:          StaticResolver(screenTypeNumerixConfig, modulePromote, serviceNumerix),
		resolverNumerixConfigDiscovery:        StaticResolver(screenTypeNumerixConfig, moduleView, serviceNumerix),
		resolverNumerixConfigEdit:             StaticResolver(screenTypeNumerixConfig, moduleEdit, serviceNumerix),
		resolverNumerixConfigRequestReview:    StaticResolver(screenTypeNumerixConfigApproval, moduleReview, serviceNumerix),
		resolverNumerixConfigRequestCancel:    StaticResolver(screenTypeNumerixConfigApproval, moduleCancel, serviceNumerix),
		resolverNumerixConfigRequestDiscovery: StaticResolver(screenTypeNumerixConfigApproval, moduleView, serviceNumerix),
		resolverNumerixConfigRequestDelete:    StaticResolver(screenTypeNumerixConfigApproval, moduleDelete, serviceNumerix),
		resolverNumerixTestGenerateRequest:    StaticResolver(screenTypeNumerixConfigTesting, moduleTest, serviceNumerix),
		resolverNumerixTestExecuteRequest:     StaticResolver(screenTypeNumerixConfigTesting, moduleTest, serviceNumerix),
	}
}
