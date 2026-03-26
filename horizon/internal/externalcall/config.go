package externalcall

// Config represents the configuration structure for deployable resources
// This type is kept for backward compatibility with existing code
type Config struct {
	MinReplica    string `json:"min_replica"`
	MaxReplica    string `json:"max_replica"`
	RunningStatus string `json:"running_status"`
}
