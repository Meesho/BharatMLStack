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
<<<<<<< HEAD:horizon/internal/middlewares/resolver/resolver_registry.go
		NewnumerixServiceResolver,
		NewPredatorServiceResolver,
		NewInferFlowServiceResolver,
		NewEmbeddingPlatformServiceResolver,
		NewOnlineFeatureStoreResolver,
=======
		NewPredatorServiceResolver,
		NewDeployableServiceResolver,
		NewInferflowServiceResolver,
		NewNumerixServiceResolver,
		NewApplicationServiceResolver,
		NewSkyeServiceResolver,
>>>>>>> 719e1f68b6c4710e883a4d61b281c16133c167a5:horizon/internal/middleware/resolver/resolver_registry.go
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
