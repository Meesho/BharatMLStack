package onfs

import "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/onfs/retrieve"

type Config struct {
	Host        string
	Port        string
	DeadLine    int
	PlainText   bool
	BatchSize   int
	CallerId    string
	CallerToken string
}

// Response represents the server response
type Response struct {
	Err  string           `json:"err"`
	Resp *retrieve.Result `json:"resp"`
}

type DecodedResponse struct {
	Err  string                  `json:"err"`
	Resp *retrieve.DecodedResult `json:"resp"`
}

// Query represents the feature retrieval query
type Query struct {
	EntityLabel   string         `json:"entity-label"`
	FeatureGroups []FeatureGroup `json:"feature-groups"`
	KeysSchema    []string       `json:"keys-schema"`
	Keys          []Keys         `json:"keys"`
}

// Keys represents a collection of key columns
type Keys struct {
	Cols []string `json:"cols"`
}

// FeatureGroup represents a group of features
type FeatureGroup struct {
	Label         string   `json:"label"`
	FeatureLabels []string `json:"feature-labels"`
}

// Feature represents a single feature with its column index
type Feature struct {
	Label     string `json:"label"`
	ColumnIdx int32
}

// FeatureSchema represents the schema for a feature group
type FeatureSchema struct {
	FeatureGroupLabel string    `json:"feature-group-label"`
	Features          []Feature `json:"features"`
}

// Row represents a single row of data
type Row struct {
	Keys    []string `json:"keys"`
	Columns [][]byte `json:"columns"`
}

// Result represents the complete feature retrieval result
type Result struct {
	EntityLabel    string          `json:"entity-label"`
	KeysSchema     []string        `json:"keys-schema"`
	FeatureSchemas []FeatureSchema `json:"feature-schemas"`
	Rows           []Row           `json:"rows"`
}

// DecodedRow represents a row with decoded string values
type DecodedRow struct {
	Keys    []string `json:"keys"`
	Columns []string `json:"columns"`
}

// DecodedResult represents the complete decoded feature retrieval result
type DecodedResult struct {
	KeysSchema     []string        `json:"keys-schema"`
	FeatureSchemas []FeatureSchema `json:"feature-schemas"`
	Rows           []DecodedRow    `json:"rows"`
}

// EntityBulkPayload represents the bulk response for multiple entities
type EntityBulkPayload struct {
	Label          string
	EntityPayloads []EntityPayload
}

// EntityPayload represents the features for a single entity
type EntityPayload struct {
	EntityIds            []EntityId            `json:"entity-ids"`
	FeatureGroupPayloads []FeatureGroupPayload `json:"feature-group-payloads"`
}

// EntityId represents the identifier for an entity
type EntityId struct {
	EntityKeys []EntityKey `json:"entity-keys"`
}

// EntityKey represents a single key-value pair for entity identification
type EntityKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// FeatureGroupPayload represents the features for a specific group
type FeatureGroupPayload struct {
	Label           string           `json:"label"`
	FeaturePayloads []FeaturePayload `json:"feature-payloads"`
}

// FeaturePayload represents a single feature value
type FeaturePayload struct {
	Label string `json:"label"`
	Value []byte `json:"value"`
}

// PersistFeaturesRequest represents the request for persisting features
type PersistFeaturesRequest struct {
	EntityLabel   string               `json:"entity-label"`
	KeysSchema    []string             `json:"keys-schema"`
	FeatureGroups []FeatureGroupSchema `json:"feature-groups"`
	Data          []Data               `json:"data"`
}

// FeatureGroupSchema represents the schema for a feature group
type FeatureGroupSchema struct {
	Label         string   `json:"label"`
	FeatureLabels []string `json:"feature-labels"`
}

// Data represents a single data entry with keys and feature values
type Data struct {
	KeyValues     []string        `json:"key-values"`
	FeatureValues []FeatureValues `json:"feature-values"`
}

// FeatureValues represents the values for a feature
type FeatureValues struct {
	Values Values `json:"values"`
}

// Values represents different types of feature values
type Values struct {
	Fp32Values   []float64 `json:"fp32-values"`
	Fp64Values   []float64 `json:"fp64-values"`
	Int32Values  []int32   `json:"int32-values"`
	Int64Values  []int64   `json:"int64-values"`
	Uint32Values []uint32  `json:"uint32-values"`
	Uint64Values []uint64  `json:"uint64-values"`
	StringValues []string  `json:"string-values"`
	BoolValues   []bool    `json:"bool-values"`
	Vector       []Vector  `json:"vector"`
}

// Vector represents a vector of values
type Vector struct {
	Values Values `json:"values"`
}

// PersistFeaturesResponse represents the response from persisting features
type PersistFeaturesResponse struct {
	Message string `json:"message"`
}
