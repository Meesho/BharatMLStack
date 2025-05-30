package listeners

type FeatureDataEvent struct {
	TimeStamp          int64          `json:"timestamp"`
	EntityLabel        string         `json:"entity_label"`
	KeysSchema         []string       `json:"keys_schema"`
	FeatureGroupSchema []FeatureGroup `json:"feature_group_schema"`
	Value              []Value        `json:"value"`
}

type FeatureGroup struct {
	Label         string   `json:"label"`
	FeatureLabels []string `json:"feature_labels"`
}

type FeatureValue struct {
	Values []string `json:"values"`
}

type Value struct {
	KeyValues     []string       `json:"key_values"`
	FeatureValues []FeatureValue `json:"feature_values"`
}
