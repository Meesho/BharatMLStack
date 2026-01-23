// middlewares/handler.go
package resolver

import (
	"log"
)

type Handler struct {
	ResolverRegistry map[string]Func
}

func NewHandler() (*Handler, error) {
	registry := make(map[string]Func)
	resolverList := []func() (ServiceResolver, error){
		NewPredatorServiceResolver,
		NewDeployableServiceResolver,
		NewInferflowServiceResolver,
		NewNumerixServiceResolver,
		NewApplicationServiceResolver,
		NewSkyeServiceResolver,
	}

	for _, rFn := range resolverList {
		resolver, err := rFn()
		if err != nil {
			log.Printf("error initializing resolver: %v", err)
			return nil, err
		}
		for k, v := range resolver.GetResolvers() {
			registry[k] = v
		}
	}
	return &Handler{
		ResolverRegistry: registry,
	}, nil
}
