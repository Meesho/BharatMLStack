package inferflow

import (
	"sync"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/components"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
	"github.com/Meesho/dag-topology-executor/handlers/dag"
)

const featureInitComponent = "feature_initializer"

type ComponentProviderHandler struct {
	componentMap map[string]dag.AbstractComponent
	mapMutex     sync.RWMutex // To synchronize access to the map
}

func (cp *ComponentProviderHandler) RegisterComponent(request interface{}) {

	modelConfig, ok := request.(*config.ModelConfig)
	if ok {

		cp.mapMutex.Lock() // Lock for write access
		defer cp.mapMutex.Unlock()

		// populate feature initializer component
		cp.componentMap[featureInitComponent] = &components.FeatureInitComponent{
			ComponentName: featureInitComponent,
		}

		if modelConfig != nil && len(modelConfig.ConfigMap) > 0 {
			for _, value := range modelConfig.ConfigMap {
				componentConfig := value.ComponentConfig
				if !utils.IsNilOrEmpty(componentConfig) {
					// populate feature component
					fCompMap := componentConfig.FeatureComponentConfig
					if fCompMap.Size() > 0 {
						for _, k := range fCompMap.Keys() {
							cp.componentMap[k.(string)] = &components.FeatureComponent{
								ComponentName: k.(string),
							}
						}
					}

					// populate real time pricing feature component
					realTimePricingFeatureCompMap := componentConfig.RealTimePricingFeatureComponentConfig
					if !utils.IsNilOrEmpty(realTimePricingFeatureCompMap) {
						for _, k := range realTimePricingFeatureCompMap.Keys() {
							cp.componentMap[k.(string)] = &components.RealTimePricingFeatureComponent{
								ComponentName: k.(string),
							}
						}
					}

					// populate ranker component
					pCompMap := componentConfig.PredatorComponentConfig
					if pCompMap.Size() > 0 {
						for _, k := range pCompMap.Keys() {
							cp.componentMap[k.(string)] = &components.PredatorComponent{
								ComponentName: k.(string),
							}
						}
					}
					//populate numerix component
					gCompMap := componentConfig.NumerixComponentConfig
					if gCompMap.Size() > 0 {
						for _, k := range gCompMap.Keys() {
							cp.componentMap[k.(string)] = &components.NumerixComponent{
								ComponentName: k.(string),
							}
						}
					}
				}
			}
		}
	}
}

func (cp *ComponentProviderHandler) GetComponent(componentName string) dag.AbstractComponent {
	cp.mapMutex.RLock() // Lock for read access
	defer cp.mapMutex.RUnlock()
	return cp.componentMap[componentName]
}
