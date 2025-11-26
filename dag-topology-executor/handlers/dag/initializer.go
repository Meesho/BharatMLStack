package dag

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/dag-topology-executor/internal/errors"
	"github.com/Meesho/BharatMLStack/dag-topology-executor/pkg/logger"
	"github.com/Meesho/BharatMLStack/dag-topology-executor/pkg/utils"
)

type ComponentInitializer struct {
	ComponentProvider ComponentProvider
}

func (ci *ComponentInitializer) InitializeComponentDag(dagConfigs map[string][]string) (*DagTopology, error) {

	if !utils.IsValidDag(dagConfigs) {
		return nil, &errors.InvalidDagError{ErrorMsg: fmt.Sprintf("Invalid DAG: %v", dagConfigs)}
	}
	adjacencyListMap := getComponentAdjacencyMap(dagConfigs)
	zeroInDegreeComponents := getZeroInDegreeComponents(adjacencyListMap)
	componentMap, err := ci.getAbstractComponents(zeroInDegreeComponents, dagConfigs)
	if err != nil {
		logger.Panic("Error initializing dag!", err)
		return nil, err
	}
	inDegreeMap := getInDegreeMap(adjacencyListMap)
	return &DagTopology{
		ZeroInDegreeComponents:    zeroInDegreeComponents,
		AbstractComponentMap:      componentMap,
		ComponentInDegreeMap:      inDegreeMap,
		ComponentAdjacencyListMap: adjacencyListMap,
	}, nil
}

func (ci *ComponentInitializer) getAbstractComponents(zeroInDegreeComponents []string, dagConfigs map[string][]string) (map[string]AbstractComponent, error) {

	componentMap := make(map[string]AbstractComponent)
	nodes := make(map[string]bool)
	for _, node := range zeroInDegreeComponents {
		nodes[node] = true
	}
	for parentNode := range dagConfigs {
		nodes[parentNode] = true
	}
	for node := range nodes {
		currentComponent := ci.ComponentProvider.GetComponent(node)
		if currentComponent == nil {
			return nil, &errors.InvalidComponentError{ErrorMsg: "Concrete Implementation not present for component: " + node}
		}
		componentMap[node] = currentComponent
	}
	return componentMap, nil
}

func getInDegreeMap(adjacencyListMap map[string]map[string]bool) map[string]int {

	inDegreeMap := make(map[string]int)
	for _, nodes := range adjacencyListMap {
		for node := range nodes {
			inDegreeMap[node]++
		}
	}
	return inDegreeMap
}

func getZeroInDegreeComponents(adjacencyListMap map[string]map[string]bool) []string {

	zeroInDegreeNodes := make(map[string]bool)
	nonZeroInDegreeNodes := make(map[string]bool)
	for _, nodes := range adjacencyListMap {
		for node := range nodes {
			nonZeroInDegreeNodes[node] = true
		}
	}
	for parentNode := range adjacencyListMap {
		if _, exists := nonZeroInDegreeNodes[parentNode]; !exists {
			zeroInDegreeNodes[parentNode] = true
		}
	}
	result := make([]string, 0, len(zeroInDegreeNodes))
	for node := range zeroInDegreeNodes {
		result = append(result, node)
	}
	return result
}

func getComponentAdjacencyMap(dagConfigs map[string][]string) map[string]map[string]bool {

	adjacencyMap := make(map[string]map[string]bool)
	for parentNode, childNodes := range dagConfigs {
		for _, childNode := range childNodes {
			if _, exists := adjacencyMap[childNode]; !exists {
				adjacencyMap[childNode] = make(map[string]bool)
			}
			adjacencyMap[childNode][parentNode] = true
		}
	}
	return adjacencyMap
}
