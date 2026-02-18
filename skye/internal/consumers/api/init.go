package api

import (
	"github.com/Meesho/BharatMLStack/skye/pkg/httpframework"
)

const (
	HeathCheckPath = "/health"
)

func Init() {
	httpframework.Instance().GET(HeathCheckPath, healthProvider)
}
