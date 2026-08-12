package profiling

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func runtimeRule(m RuntimeMetric) (collectors.GoRuntimeMetricsRule, error) {
	switch m {
	case RuntimeMetricMemory:
		return collectors.MetricsMemory, nil
	case RuntimeMetricGC:
		return collectors.MetricsGC, nil
	case RuntimeMetricScheduler:
		return collectors.MetricsScheduler, nil
	case RuntimeMetricAll:
		return collectors.MetricsAll, nil
	default:
		return collectors.GoRuntimeMetricsRule{}, fmt.Errorf("unknown runtime metric %d", m)
	}
}

// registerRuntimeMetrics exports Go's runtime/metrics through the standard
// Prometheus Go collector, which is what produces the go_* series the
// go-runtime-metrics dashboard reads.
// The parameter is NOT called `metrics`: inferflow's metric package is
// `pkg/metrics` (plural, unlike the other modules' `pkg/metric`), so a parameter
// of that name shadows the import and `metrics.Registerer()` below silently
// resolves to the slice instead of the package.
func registerRuntimeMetrics(selected []RuntimeMetric) error {
	rules := make([]collectors.GoRuntimeMetricsRule, 0, len(selected))
	for _, m := range selected {
		rule, err := runtimeRule(m)
		if err != nil {
			return err
		}
		rules = append(rules, rule)
	}
	collector := collectors.NewGoCollector(collectors.WithGoCollectorRuntimeMetrics(rules...))
	return metrics.Registerer().Register(collector)
}
