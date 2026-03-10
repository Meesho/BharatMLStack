package application

import "sort"

func computeRangeAssignment(partitionCount int, members []string, targetConsumer string) []int {
	if partitionCount <= 0 || len(members) == 0 || targetConsumer == "" {
		return nil
	}
	sorted := append([]string(nil), members...)
	sort.Strings(sorted)

	index := -1
	for i, member := range sorted {
		if member == targetConsumer {
			index = i
			break
		}
	}
	if index < 0 {
		return nil
	}

	n := len(sorted)
	base := partitionCount / n
	remainder := partitionCount % n

	start := 1
	for i := 0; i < index; i++ {
		width := base
		if i < remainder {
			width++
		}
		start += width
	}
	width := base
	if index < remainder {
		width++
	}
	if width <= 0 {
		return nil
	}

	assigned := make([]int, 0, width)
	for part := start; part < start+width; part++ {
		assigned = append(assigned, part)
	}
	return assigned
}
