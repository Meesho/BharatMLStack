package etcd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/spf13/viper"
)

type Etcd struct {
	instance etcd.Etcd
	appName  string
	env      string
}

func NewEtcdInstance() *Etcd {
	return &Etcd{
		instance: etcd.Instance()[viper.GetString("NUMERIX_APP_NAME")],
		appName:  viper.GetString("NUMERIX_APP_NAME"),
		env:      viper.GetString("APP_ENV"),
	}
}

func (e *Etcd) CreateConfig(configId string, expression string) error {
	configMap := make(map[string]interface{})
	configMap["expression"] = expression

	// Use JSON encoder with HTML escaping disabled
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(configMap)
	if err != nil {
		return err
	}

	// Remove the trailing newline added by Encode
	configJson := bytes.TrimSpace(buf.Bytes())

	return e.instance.CreateNode(fmt.Sprintf("/config/%s/expression-config/%s", e.appName, configId), string(configJson))
}

func (e *Etcd) UpdateConfig(configId string, expression string) error {
	configMap := make(map[string]interface{})
	configMap["expression"] = expression

	// Use JSON encoder with HTML escaping disabled
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(configMap)
	if err != nil {
		return err
	}

	configJson := bytes.TrimSpace(buf.Bytes())

	return e.instance.SetValue(fmt.Sprintf("/config/%s/expression-config/%s", e.appName, configId), string(configJson))
}
