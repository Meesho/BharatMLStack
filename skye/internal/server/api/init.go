package api

import "github.com/Meesho/BharatMLStack/skye/pkg/httpframework"

const (
	healthCheckPath = "/health"
)

func Init() {
	httpframework.Instance().GET(healthCheckPath, healthProvider)
}
