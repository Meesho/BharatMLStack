package asyncloguploader

import (
	"os"
	"strings"
	"time"

	"github.com/Meesho/go-core/metric"
)

const renameRetryCount = 3
const renameRetryDelay = 10 * time.Millisecond

// renameTmpToLog renames a .log.tmp file to .log with retries.
// On failure after retries, emits MetricFileRenameFailed and returns nil (ignore per spec).
func renameTmpToLog(tmpPath string, metricTags []string) {
	logPath := strings.TrimSuffix(tmpPath, ".tmp")
	if logPath == tmpPath {
		return // not a .tmp file, nothing to do
	}
	for i := 0; i < renameRetryCount; i++ {
		if err := os.Rename(tmpPath, logPath); err == nil {
			return
		}
		time.Sleep(renameRetryDelay)
	}
	metric.Incr(MetricFileRenameFailed, getBaseTags(metricTags))
}
