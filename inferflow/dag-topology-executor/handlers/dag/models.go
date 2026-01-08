package dag

type DagTopology struct {
	ComponentAdjacencyListMap map[string]map[string]bool
	ComponentInDegreeMap      map[string]int
	ZeroInDegreeComponents    []string
	AbstractComponentMap      map[string]AbstractComponent
}

type AbstractComponent interface {
	Run(request interface{})
	GetComponentName() string
}

type ComponentProvider interface {
	GetComponent(string) AbstractComponent
	RegisterComponent(request interface{})
}
