package utils

func IsValidDag(dagConfig map[string][]string) bool {
	return !isCycleInDag(dagConfig)
}

func isCycleInDag(dagConfig map[string][]string) bool {

	nodeToInDegree := make(map[string]int)
	for parentNode := range dagConfig {
		nodeToInDegree[parentNode] = 0
	}
	for _, childNodes := range dagConfig {
		for _, v := range childNodes {
			nodeToInDegree[v]++
		}
	}

	queue := make([]string, 0)
	for node := range nodeToInDegree {
		if nodeToInDegree[node] == 0 {
			queue = append(queue, node)
		}
	}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, v := range dagConfig[u] {
			nodeToInDegree[v]--
			if nodeToInDegree[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	for _, inDegree := range nodeToInDegree {
		if inDegree != 0 {
			return true
		}
	}

	return false
}
