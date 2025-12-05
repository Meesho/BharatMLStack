package resolver

const (
	screenTypeApplicationDiscoveryRegistry = "application-discovery-registry"
	screenTypeConnectionConfig             = "connection-config"

	serviceApplication = "application"

	resolverApplicationDiscovery      = "ApplicationDiscoveryRegistryResolver"
	resolverApplicationOnboard        = "ApplicationOnboardResolver"
	resolverApplicationEdit           = "ApplicationEditResolver"
	resolverConnectionConfigDiscovery = "ConnectionConfigDiscoveryResolver"
	resolverConnectionConfigOnboard   = "ConnectionConfigOnboardResolver"
	resolverConnectionConfigEdit      = "ConnectionConfigEditResolver"
)

type ApplicationResolver struct {
}

func NewApplicationServiceResolver() (ServiceResolver, error) {
	return &ApplicationResolver{}, nil
}

func (r *ApplicationResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		resolverApplicationDiscovery:      StaticResolver(screenTypeApplicationDiscoveryRegistry, moduleView, serviceApplication),
		resolverApplicationOnboard:        StaticResolver(screenTypeApplicationDiscoveryRegistry, moduleOnboard, serviceApplication),
		resolverApplicationEdit:           StaticResolver(screenTypeApplicationDiscoveryRegistry, moduleEdit, serviceApplication),
		resolverConnectionConfigDiscovery: StaticResolver(screenTypeConnectionConfig, moduleView, serviceApplication),
		resolverConnectionConfigOnboard:   StaticResolver(screenTypeConnectionConfig, moduleOnboard, serviceApplication),
		resolverConnectionConfigEdit:      StaticResolver(screenTypeConnectionConfig, moduleEdit, serviceApplication),
	}
}
