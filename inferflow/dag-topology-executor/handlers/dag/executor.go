package dag

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/inferflow/dag-topology-executor/internal/errors"
	"github.com/Meesho/BharatMLStack/inferflow/dag-topology-executor/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/dag-topology-executor/pkg/utils"
)

const (
	TERMINAL_NODE string = "terminal_node"
)

type Executor struct {
	Registry *ComponentGraphRegistry
}

func (e *Executor) Execute(dagConfig map[string][]string, request interface{}) {

	if utils.IsNilOrEmpty(request) || utils.IsNilOrEmpty(dagConfig) {
		logger.Panic("Bad Request :: ",
			&errors.BadRequestError{ErrorMsg: "Either dag config or exector request is invalid"})
		return
	}
	dagTopology, err := e.Registry.getDagTopology(dagConfig)

	if err != nil {
		logger.Panic("Error initializing dag!", err)
		return
	}

	executeDag(*dagTopology, request)
}

func executeDag(dagTopology DagTopology, request interface{}) {

	inDegreeMap := utils.NewConcurrentMap()
	for key, value := range dagTopology.ComponentInDegreeMap {
		inDegreeMap.Set(key, value)
	}
	totalCompCount := uint64(inDegreeMap.Size() + 1)
	componentMap := dagTopology.AbstractComponentMap
	adjacencyListMap := dagTopology.ComponentAdjacencyListMap
	zeroInDegreeComponents := dagTopology.ZeroInDegreeComponents

	queue := make(chan string, len(zeroInDegreeComponents))
	for _, component := range zeroInDegreeComponents {
		queue <- component
	}
	var queueOpen bool = true
	var executedCompCount uint64 = 0
	for {
		component, ok := <-queue
		if !ok {
			queueOpen = false
			// queue is already closed
			break
		}
		if component == TERMINAL_NODE {
			executedCompCount++
			if executedCompCount == totalCompCount {
				queueOpen = false
				close(queue)
				break
			}
		} else {
			go func(component string) {
				abstractComponent, ok := componentMap[component]
				if !ok {
					logger.Error("Error executing DAG component ",
						&errors.InvalidComponentError{ErrorMsg: fmt.Sprintf("Concrete Implementation not present for component: %s", component)})
					return
				}
				abstractComponent.Run(request)
				inDegreeMap.Delete(component)
				if adjacencyList, ok := adjacencyListMap[component]; ok {
					for c := range adjacencyList {
						if value, ok := inDegreeMap.UpdateAndGet(c); ok {
							if value == 0 {
								if queueOpen {
									queue <- c
								} else {
									logger.Error("Queue is closed while inserting component node", nil)
								}
							}
						}
					}
				}
				if queueOpen {
					queue <- TERMINAL_NODE
				} else {
					logger.Error("Queue is closed while inserting terminal node", nil)
				}
			}(component)
		}
	}
}
