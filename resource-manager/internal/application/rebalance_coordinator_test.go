package application

import (
	"reflect"
	"testing"
)

func TestComputeRangeAssignment(t *testing.T) {
	tests := []struct {
		name           string
		partitionCount int
		members        []string
		consumer       string
		want           []int
	}{
		{
			name:           "8 partitions and 2 members first consumer",
			partitionCount: 8,
			members:        []string{"consumer-1", "consumer-2"},
			consumer:       "consumer-1",
			want:           []int{1, 2, 3, 4},
		},
		{
			name:           "8 partitions and 2 members second consumer",
			partitionCount: 8,
			members:        []string{"consumer-1", "consumer-2"},
			consumer:       "consumer-2",
			want:           []int{5, 6, 7, 8},
		},
		{
			name:           "8 partitions and 3 members middle consumer",
			partitionCount: 8,
			members:        []string{"consumer-1", "consumer-2", "consumer-3"},
			consumer:       "consumer-2",
			want:           []int{4, 5, 6},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeRangeAssignment(tc.partitionCount, tc.members, tc.consumer)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
