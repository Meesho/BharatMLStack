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
	Err  string
	Resp *retrieve.Result
}

type DecodedResponse struct {
	Err  string
	Resp *retrieve.DecodedResult
}

// Query represents the feature retrieval query
type Query struct {
	EntityLabel   string
	FeatureGroups []FeatureGroup
	KeysSchema    []string
	Keys          []Keys
}

// Keys represents a collection of key columns
type Keys struct {
	Cols []string
}

// FeatureGroup represents a group of features
type FeatureGroup struct {
	Label         string
	FeatureLabels []string
}

// Feature represents a single feature with its column index
type Feature struct {
	Label     string
	ColumnIdx int32
}

// FeatureSchema represents the schema for a feature group
type FeatureSchema struct {
	FeatureGroupLabel string
	Features          []Feature
}

// Row represents a single row of data
type Row struct {
	Keys    []string
	Columns [][]byte
}

// Result represents the complete feature retrieval result
type Result struct {
	EntityLabel    string
	KeysSchema     []string
	FeatureSchemas []FeatureSchema
	Rows           []Row
}

// DecodedRow represents a row with decoded string values
type DecodedRow struct {
	Keys    []string
	Columns []string
}

// DecodedResult represents the complete decoded feature retrieval result
type DecodedResult struct {
	KeysSchema     []string
	FeatureSchemas []FeatureSchema
	Rows           []DecodedRow
}

// EntityBulkPayload represents the bulk response for multiple entities
type EntityBulkPayload struct {
	Label          string
	EntityPayloads []EntityPayload
}

// EntityPayload represents the features for a single entity
type EntityPayload struct {
	EntityIds            []EntityId
	FeatureGroupPayloads []FeatureGroupPayload
}

// EntityId represents the identifier for an entity
type EntityId struct {
	EntityKeys []EntityKey
}

// EntityKey represents a single key-value pair for entity identification
type EntityKey struct {
	Type  string
	Value string
}

// FeatureGroupPayload represents the features for a specific group
type FeatureGroupPayload struct {
	Label           string
	FeaturePayloads []FeaturePayload
}

// FeaturePayload represents a single feature value
type FeaturePayload struct {
	Label string
	Value []byte
}

// PersistFeaturesRequest represents the request for persisting features
type PersistFeaturesRequest struct {
	EntityLabel   string
	KeysSchema    []string
	FeatureGroups []FeatureGroupSchema
	Data          []Data
}

// FeatureGroupSchema represents the schema for a feature group
type FeatureGroupSchema struct {
	Label         string
	FeatureLabels []string
}

// Data represents a single data entry with keys and feature values
type Data struct {
	KeyValues     []string
	FeatureValues []FeatureValues
}

// FeatureValues represents the values for a feature
type FeatureValues struct {
	Values Values
}

// Values represents different types of feature values
type Values struct {
	Fp32Values   []float64
	Fp64Values   []float64
	Int32Values  []int32
	Int64Values  []int64
	Uint32Values []uint32
	Uint64Values []uint64
	StringValues []string
	BoolValues   []bool
	Vector       []Vector
}

// Vector represents a vector of values
type Vector struct {
	Values Values
}

// PersistFeaturesResponse represents the response from persisting features
type PersistFeaturesResponse struct {
	Message string
}
