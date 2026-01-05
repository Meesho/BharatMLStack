package health

import "github.com/Meesho/go-core/httpframework"

const (
	healthCheckPath = "/health"
)

func Init() {
	httpframework.Instance().GET(healthCheckPath, Health)
}
