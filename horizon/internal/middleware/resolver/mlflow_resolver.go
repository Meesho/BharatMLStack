package resolver

const (
	screenTypeMLFlow = "mlflow"
	serviceMLFlow    = "mlflow"
	moduleProxy      = "proxy"

	// Resolvers
	resolverMLFlowProxy = "MLFlowProxyResolver"
)

type MLFlowResolver struct{}

func NewMLFlowServiceResolver() (ServiceResolver, error) {
	return &MLFlowResolver{}, nil
}

func (m *MLFlowResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		resolverMLFlowProxy: StaticResolver(screenTypeMLFlow, moduleProxy, serviceMLFlow),
	}
}
