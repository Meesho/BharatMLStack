package etcd

type NumerixConfig struct {
	Expression string `json:"expression"`
}

type NumerixConfigRegistery struct {
	ExpressionConfig map[string]NumerixConfig `json:"expression-config"`
}
