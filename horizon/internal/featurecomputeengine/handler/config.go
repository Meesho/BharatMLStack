package handler

// Config holds configuration for the feature compute engine handler.
type Config struct {
	ArtifactBasePath string // e.g. "s3://bucket/artifacts" or "/_artifacts"
	ServingBasePath  string // e.g. "s3://bucket/serving" or "/serving"
	DefaultPartition string // e.g. "ds"
}
