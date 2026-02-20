package config

import "github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"

type EtcdRegistry struct {
	ShadowDeployables map[string]map[string]models.ShadowDeployable `json:"shadow-deployables"`
}
