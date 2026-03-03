package handler

// --- Necessity ---

type Necessity string

const (
	NecessityActive    Necessity = "active"
	NecessityTransient Necessity = "transient"
	NecessitySkipped   Necessity = "skipped"
)

// --- In-memory DAG ---

type AssetNode struct {
	AssetName    string
	EntityName   string
	EntityKey    string
	Notebook     string
	TriggerType  string
	Schedule     *string
	Serving      bool
	AssetVersion string
}

type DAGEdge struct {
	From       string
	To         string
	InputType  string
	WindowSize *int
}

type LineageDAG struct {
	Nodes    map[string]AssetNode
	Parents  map[string][]string
	Children map[string][]string
	Edges    []DAGEdge
}

// --- Execution Plan (sent to Databricks via Airflow) ---

type AssetAction string

const (
	ActionExecute         AssetAction = "execute"
	ActionSkipCached      AssetAction = "skip_cached"
	ActionSkipUnnecessary AssetAction = "skip_unnecessary"
)

type AssetExecutionPlan struct {
	AssetName        string            `json:"asset_name"`
	Action           AssetAction       `json:"action"`
	Necessity        Necessity         `json:"necessity"`
	ComputeKey       string            `json:"compute_key,omitempty"`
	InputBindings    map[string]string `json:"input_bindings,omitempty"`
	ArtifactPath     string            `json:"artifact_path,omitempty"`
	ExistingArtifact string            `json:"existing_artifact,omitempty"`
	Reason           string            `json:"reason,omitempty"`
}

type NotebookExecutionPlan struct {
	Notebook     string               `json:"notebook"`
	TriggerType  string               `json:"trigger_type"`
	Partition    string               `json:"partition"`
	Assets       []AssetExecutionPlan `json:"assets"`
	SharedInputs map[string]string    `json:"shared_inputs"`
}

// --- Plan Stage (diff/impact analysis) ---

type AssetChange struct {
	AssetName     string   `json:"asset_name"`
	ChangeType    string   `json:"change_type"`
	InputsAdded   []string `json:"inputs_added,omitempty"`
	InputsRemoved []string `json:"inputs_removed,omitempty"`
	CodeChanged   bool     `json:"code_changed"`
}

type RecomputationImpact struct {
	AssetName          string    `json:"asset_name"`
	Reason             string    `json:"reason"`
	TriggeredBy        string    `json:"triggered_by,omitempty"`
	PartitionsAffected int       `json:"partitions_affected"`
	PartitionRange     [2]string `json:"partition_range"`
	EstimatedCostUSD   *float64  `json:"estimated_cost_usd,omitempty"`
}

type PlanResult struct {
	PlanID             string                `json:"plan_id"`
	Changes            []AssetChange         `json:"changes"`
	NecessityChanges   map[string][2]string  `json:"necessity_changes"`
	DirectImpact       []RecomputationImpact `json:"direct_impact"`
	CascadeImpact      []RecomputationImpact `json:"cascade_impact"`
	TotalPartitions    int                   `json:"total_partitions"`
	TotalEstimatedCost *float64              `json:"total_estimated_cost_usd,omitempty"`
	Warnings           []string              `json:"warnings,omitempty"`
}

// --- API Request/Response Models ---

type RegisterAssetsRequest struct {
	Assets []AssetSpecPayload `json:"assets" validate:"required,min=1"`
}

type AssetSpecPayload struct {
	Name        string         `json:"name" validate:"required"`
	Version     string         `json:"version" validate:"required"`
	Entity      string         `json:"entity" validate:"required"`
	EntityKey   string         `json:"entity_key" validate:"required"`
	Notebook    string         `json:"notebook" validate:"required"`
	Partition   string         `json:"partition" validate:"required"`
	Trigger     string         `json:"trigger" validate:"required"`
	Schedule    *string        `json:"schedule,omitempty"`
	Serving     bool           `json:"serving"`
	Incremental bool           `json:"incremental"`
	Freshness   *string        `json:"freshness,omitempty"`
	Inputs      []InputPayload `json:"inputs"`
	Checks      []string       `json:"checks,omitempty"`
}

type InputPayload struct {
	Name       string `json:"name" validate:"required"`
	Partition  string `json:"partition" validate:"required"`
	Type       string `json:"type" validate:"required,oneof=internal external"`
	WindowSize *int   `json:"window_size,omitempty"`
}

type GetExecutionPlanRequest struct {
	Notebook    string `json:"notebook" validate:"required"`
	TriggerType string `json:"trigger_type" validate:"required"`
	Partition   string `json:"partition" validate:"required"`
}

type ReportAssetReadyRequest struct {
	AssetName    string `json:"asset_name" validate:"required"`
	Partition    string `json:"partition" validate:"required"`
	ComputeKey   string `json:"compute_key" validate:"required"`
	ArtifactPath string `json:"artifact_path" validate:"required"`
	DeltaVersion int64  `json:"delta_version"`
}

type SetServingOverrideRequest struct {
	AssetName string `json:"asset_name" validate:"required"`
	Serving   bool   `json:"serving"`
	Reason    string `json:"reason" validate:"required"`
	UpdatedBy string `json:"updated_by" validate:"required"`
}

type ProposePlanRequest struct {
	Assets []AssetSpecPayload `json:"assets" validate:"required"`
	DryRun bool               `json:"dry_run"`
}
