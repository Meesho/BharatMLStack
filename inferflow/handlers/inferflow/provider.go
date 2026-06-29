//go:build !meesho

package inferflow

import (
	"sync"

	"github.com/Meesho/BharatMLStack/inferflow/dag-topology-executor/handlers/dag"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
)

const featureInitComponent = "feature_initializer"

type ComponentProviderHandler struct {
	componentMap map[string]dag.AbstractComponent
	mapMutex     sync.RWMutex // To synchronize access to the map
}

// RegisterComponent rebuilds the component map from the supplied ModelConfig
// and swaps it atomically into place. It is registered as an etcd config-change
// callback (see cmd/inferflow/main.go), so it runs once per config event.
//
// Prior versions APPENDED to the existing componentMap without ever pruning,
// which caused the map to grow unboundedly across config events — entries
// from removed or renamed models accumulated for the pod's lifetime. The
// rebuild-and-swap pattern below mirrors the one already used for the
// feature-schema cache (handlers/config/config_schema.go), so the cache
// reflects exactly the current model config after each event.
func (cp *ComponentProviderHandler) RegisterComponent(request interface{}) {
	modelConfig, ok := request.(*config.ModelConfig)
	if !ok {
		return
	}

	// Build a fresh map locally; never mutate cp.componentMap while readers
	// (GetComponent) may be holding the read lock. The final atomic swap
	// publishes the new map; the old map becomes GC-able.
	newMap := make(map[string]dag.AbstractComponent)

	// feature initializer component is always present
	newMap[featureInitComponent] = &components.FeatureInitComponent{
		ComponentName: featureInitComponent,
	}

	if modelConfig != nil && len(modelConfig.ConfigMap) > 0 {
		for _, value := range modelConfig.ConfigMap {
			componentConfig := value.ComponentConfig
			if utils.IsNilOrEmpty(componentConfig) {
				continue
			}

			// feature components
			if fCompMap := componentConfig.FeatureComponentConfig; fCompMap.Size() > 0 {
				for _, k := range fCompMap.Keys() {
					newMap[k.(string)] = &components.FeatureComponent{
						ComponentName: k.(string),
					}
				}
			}

			// predator (ranker) components
			if pCompMap := componentConfig.PredatorComponentConfig; pCompMap.Size() > 0 {
				for _, k := range pCompMap.Keys() {
					newMap[k.(string)] = &components.PredatorComponent{
						ComponentName: k.(string),
					}
				}
			}

			// numerix components
			if gCompMap := componentConfig.NumerixComponentConfig; gCompMap.Size() > 0 {
				for _, k := range gCompMap.Keys() {
					newMap[k.(string)] = &components.NumerixComponent{
						ComponentName: k.(string),
					}
				}
			}
		}
	}

	cp.mapMutex.Lock()
	cp.componentMap = newMap
	cp.mapMutex.Unlock()
}

func (cp *ComponentProviderHandler) GetComponent(componentName string) dag.AbstractComponent {
	cp.mapMutex.RLock() // Lock for read access
	defer cp.mapMutex.RUnlock()
	return cp.componentMap[componentName]
}
