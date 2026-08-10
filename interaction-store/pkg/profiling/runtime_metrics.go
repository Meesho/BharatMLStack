package profiling

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/interaction-store/pkg/metric"
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
func registerRuntimeMetrics(metrics []RuntimeMetric) error {
	rules := make([]collectors.GoRuntimeMetricsRule, 0, len(metrics))
	for _, m := range metrics {
		rule, err := runtimeRule(m)
		if err != nil {
			return err
		}
		rules = append(rules, rule)
	}
	collector := collectors.NewGoCollector(collectors.WithGoCollectorRuntimeMetrics(rules...))
	return metric.Registerer().Register(collector)
}
