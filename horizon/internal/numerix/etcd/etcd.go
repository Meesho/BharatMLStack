package etcd

import (
	"bytes"
	"encoding/json"
	"fmt"

	NumerixPkg "github.com/Meesho/BharatMLStack/horizon/internal/numerix"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
)

type Etcd struct {
	instance etcd.Etcd
	appName  string
	env      string
}

func NewEtcdInstance() *Etcd {
	return &Etcd{
		instance: etcd.Instance()[NumerixPkg.NumerixAppName],
		appName:  NumerixPkg.NumerixAppName,
		env:      NumerixPkg.AppEnv,
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

func (e *Etcd) DeleteConfig(configId string) error {
	return e.instance.DeleteNode(fmt.Sprintf("/config/%s/expression-config/%s", e.appName, configId))
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
