package handler

import "time"

// --- Review statuses ---

const (
	StatusPending      = "pending"
	StatusPlanComputed = "plan_computed"
	StatusApproved     = "approved"
	StatusRejected     = "rejected"
	StatusMerged       = "merged"
	StatusClosed       = "closed"
)

// --- Webhook payload (from GitHub) ---

type WebhookPayload struct {
	Action      string        `json:"action"`
	PullRequest PullRequestPL `json:"pull_request"`
	Repository  RepositoryPL  `json:"repository"`
}

type PullRequestPL struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Head    RefPL  `json:"head"`
	Base    RefPL  `json:"base"`
	User    UserPL `json:"user"`
	Merged  bool   `json:"merged"`
}

type RefPL struct {
	SHA string `json:"sha"`
	Ref string `json:"ref"`
}

type UserPL struct {
	Login string `json:"login"`
}

type RepositoryPL struct {
	FullName string `json:"full_name"`
}

// --- API request models ---

type ApproveRequest struct {
	Comment string `json:"comment"`
}

type RejectRequest struct {
	Comment string `json:"comment" validate:"required"`
}

// --- API response models ---

type ReviewResponse struct {
	ID            int64         `json:"id"`
	Repo          string        `json:"repo"`
	PRNumber      int           `json:"pr_number"`
	PRTitle       string        `json:"pr_title"`
	PRAuthor      string        `json:"pr_author"`
	PRURL         string        `json:"pr_url"`
	PRBranch      string        `json:"pr_branch,omitempty"`
	HeadSHA       string        `json:"head_sha"`
	Status        string        `json:"status"`
	AssetSummary  *AssetSummary `json:"asset_summary,omitempty"`
	ReviewedBy    string        `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time    `json:"reviewed_at,omitempty"`
	ReviewComment string        `json:"review_comment,omitempty"`
	MergedBy      string        `json:"merged_by,omitempty"`
	MergedAt      *time.Time    `json:"merged_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type AssetSummary struct {
	Total    int `json:"total"`
	New      int `json:"new"`
	Modified int `json:"modified"`
	Removed  int `json:"removed"`
}

type ReviewListResponse struct {
	Reviews []ReviewResponse `json:"reviews"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

type ComputePlanResponse struct {
	ReviewID int64       `json:"review_id"`
	Status   string      `json:"status"`
	Plan     interface{} `json:"plan"`
}

type ApproveResponse struct {
	ReviewID   int64     `json:"review_id"`
	Status     string    `json:"status"`
	ReviewedBy string    `json:"reviewed_by"`
	ReviewedAt time.Time `json:"reviewed_at"`
	Message    string    `json:"message"`
}

type StatsResponse struct {
	Pending      int64 `json:"pending"`
	PlanComputed int64 `json:"plan_computed"`
	Approved     int64 `json:"approved"`
	Rejected     int64 `json:"rejected"`
	Merged       int64 `json:"merged"`
	Closed       int64 `json:"closed"`
}

// --- Manifest types (read from .bharatml/manifest.json) ---

type Manifest struct {
	Version  string          `json:"version"`
	Notebook string          `json:"notebook"`
	Entity   string          `json:"entity"`
	Assets   []ManifestAsset `json:"assets"`
}

type ManifestAsset struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Entity      string          `json:"entity"`
	EntityKey   string          `json:"entity_key"`
	Notebook    string          `json:"notebook"`
	Partition   string          `json:"partition"`
	Trigger     string          `json:"trigger"`
	Schedule    *string         `json:"schedule,omitempty"`
	Serving     bool            `json:"serving"`
	Incremental bool            `json:"incremental"`
	Freshness   *string         `json:"freshness,omitempty"`
	Inputs      []ManifestInput `json:"inputs"`
	Checks      []string        `json:"checks,omitempty"`
}

type ManifestInput struct {
	Name       string `json:"name"`
	Partition  string `json:"partition"`
	Type       string `json:"type"`
	WindowSize *int   `json:"window_size,omitempty"`
}
