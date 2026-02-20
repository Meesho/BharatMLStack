package etcd

import (
	"encoding/json"
	"time"

	"github.com/spf13/viper"
)

const envBootstrapLayout = "BOOTSTRAP_ETCD_LAYOUT"

func bootstrapIfEnabled(instance Etcd) {
	if !viper.GetBool(envBootstrapLayout) {
		return
	}

	v1, ok := instance.(*V1)
	if !ok {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	seeds := []map[string]interface{}{
		{
			"env":  "int",
			"name": "int-predator-g2-std-8",
			"dns":  "predator-g2-std-8.meesho.int",
			"pool": "g2-std-8",
		},
		{
			"env":  "int",
			"name": "int-predator-g2-std-16",
			"dns":  "predator-g2-std-16.meesho.int",
			"pool": "g2-std-16",
		},
		{
			"env":  "prod",
			"name": "prd-predator-g2-std-8",
			"dns":  "predator-g2-std-8.meesho.io",
			"pool": "g2-std-8",
		},
		{
			"env":  "prod",
			"name": "prd-predator-g2-std-16",
			"dns":  "predator-g2-std-16.meesho.io",
			"pool": "g2-std-16",
		},
	}

	for _, seed := range seeds {
		key := v1.basePath + "/shadow-deployables/" + seed["env"].(string) + "/" + seed["name"].(string)
		exists, err := v1.IsNodeExist(key)
		if err != nil || exists {
			continue
		}

		payload := map[string]interface{}{
			"name":            seed["name"],
			"node_pool":       seed["pool"],
			"dns":             seed["dns"],
			"state":           "FREE",
			"owner":           nil,
			"pod_count":       1,
			"last_updated_at": now,
			"version":         1,
		}
		raw, _ := json.Marshal(payload)
		_ = v1.CreateNode(key, string(raw))
	}
}
