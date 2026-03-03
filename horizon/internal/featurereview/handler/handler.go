package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/featurereview"
	ghpkg "github.com/Meesho/BharatMLStack/horizon/pkg/github"
	"github.com/google/go-github/v53/github"
	"github.com/rs/zerolog/log"
)

const manifestPath = ".bharatml/manifest.json"

func (h *Handler) VerifyWebhookSignature(payload []byte, signature string) bool {
	if h.cfg.WebhookSecret == "" {
		return true
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(h.cfg.WebhookSecret))
	mac.Write(payload)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}

func (h *Handler) IsWatchedRepo(repoFullName string) bool {
	for _, r := range h.cfg.FeatureRepos {
		if r == repoFullName {
			return true
		}
	}
	return false
}

func (h *Handler) HandleWebhook(ctx context.Context, payload WebhookPayload) error {
	repoFullName := payload.Repository.FullName
	if !h.IsWatchedRepo(repoFullName) {
		log.Info().Str("repo", repoFullName).Msg("ignoring webhook from non-watched repo")
		return nil
	}

	pr := payload.PullRequest

	switch payload.Action {
	case "opened", "synchronize":
		hasManifest := h.checkManifestExists(ctx, repoFullName, pr.Head.Ref)
		if !hasManifest {
			log.Info().Str("repo", repoFullName).Int("pr", pr.Number).
				Msg("PR does not contain manifest, ignoring")
			return nil
		}
		title := pr.Title
		author := pr.User.Login
		url := pr.HTMLURL
		branch := pr.Head.Ref
		sha := pr.Head.SHA
		_, err := h.repo.Upsert(ctx, featurereview.ReviewRow{
			Repo:     repoFullName,
			PRNumber: pr.Number,
			PRTitle:  &title,
			PRAuthor: &author,
			PRURL:    &url,
			PRBranch: &branch,
			HeadSHA:  &sha,
			Status:   StatusPending,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to upsert review record")
			return fmt.Errorf("failed to upsert review: %w", err)
		}

	case "closed":
		existing, err := h.repo.GetByRepoPR(ctx, repoFullName, pr.Number)
		if err != nil {
			log.Debug().Err(err).Msg("no existing review record for closed PR")
			return nil
		}
		if pr.Merged {
			if err := h.repo.UpdateMerge(ctx, existing.ID, pr.User.Login); err != nil {
				return fmt.Errorf("failed to update merge status: %w", err)
			}
			go h.triggerApplyFlow(repoFullName, existing.ID)
		} else {
			if err := h.repo.UpdateStatus(ctx, existing.ID, StatusClosed); err != nil {
				return fmt.Errorf("failed to update closed status: %w", err)
			}
		}
	}
	return nil
}

func (h *Handler) checkManifestExists(ctx context.Context, repoFullName, branch string) bool {
	client, err := ghpkg.GetGitHubClient()
	if err != nil {
		log.Error().Err(err).Msg("GitHub client not available")
		return false
	}
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return false
	}
	_, _, resp, err := client.Repositories.GetContents(
		ctx, parts[0], parts[1], manifestPath,
		&github.RepositoryContentGetOptions{Ref: branch},
	)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func (h *Handler) ListReviews(ctx context.Context, status, repo string, page, perPage int) (*ReviewListResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	rows, total, err := h.repo.List(ctx, status, repo, page, perPage)
	if err != nil {
		return nil, err
	}
	reviews := make([]ReviewResponse, len(rows))
	for i, row := range rows {
		reviews[i] = rowToResponse(row)
	}
	return &ReviewListResponse{
		Reviews: reviews,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func (h *Handler) GetReview(ctx context.Context, id int64) (*ReviewResponse, error) {
	row, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := rowToResponse(*row)
	return &resp, nil
}

func (h *Handler) ComputePlan(ctx context.Context, id int64) (*ComputePlanResponse, error) {
	row, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("review not found: %w", err)
	}

	manifestContent, err := h.fetchManifest(ctx, row.Repo, deref(row.PRBranch, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal([]byte(manifestContent), &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}

	planResult, err := h.callHorizonPlan(ctx, manifest)
	if err != nil {
		return nil, fmt.Errorf("horizon plan failed: %w", err)
	}

	planJSON, _ := json.Marshal(planResult)
	if err := h.repo.UpdatePlan(ctx, id, manifestContent, string(planJSON)); err != nil {
		return nil, fmt.Errorf("failed to cache plan: %w", err)
	}

	return &ComputePlanResponse{
		ReviewID: id,
		Status:   StatusPlanComputed,
		Plan:     planResult,
	}, nil
}

func (h *Handler) Approve(ctx context.Context, id int64, reviewedBy, comment string) (*ApproveResponse, error) {
	row, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("review not found: %w", err)
	}
	if row.Status != StatusPlanComputed {
		return nil, fmt.Errorf("cannot approve: plan must be computed first (current status: %s)", row.Status)
	}

	if err := h.repo.UpdateReview(ctx, id, StatusApproved, reviewedBy, comment); err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	go h.approveOnGitHub(row.Repo, row.PRNumber, comment)

	now := time.Now()
	return &ApproveResponse{
		ReviewID:   id,
		Status:     StatusApproved,
		ReviewedBy: reviewedBy,
		ReviewedAt: now,
		Message:    "PR approved on GitHub",
	}, nil
}

func (h *Handler) Reject(ctx context.Context, id int64, reviewedBy, comment string) error {
	row, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("review not found: %w", err)
	}
	if row.Status != StatusPlanComputed && row.Status != StatusPending {
		return fmt.Errorf("cannot reject: review is in status %s", row.Status)
	}
	if comment == "" {
		return fmt.Errorf("comment is required for rejection")
	}

	if err := h.repo.UpdateReview(ctx, id, StatusRejected, reviewedBy, comment); err != nil {
		return fmt.Errorf("failed to update review: %w", err)
	}

	go h.commentOnGitHub(row.Repo, row.PRNumber,
		fmt.Sprintf("Feature review rejected by %s: %s", reviewedBy, comment))

	return nil
}

func (h *Handler) MergePR(ctx context.Context, id int64, mergedBy string) error {
	row, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("review not found: %w", err)
	}
	if row.Status != StatusApproved {
		return fmt.Errorf("cannot merge: review must be approved first (current status: %s)", row.Status)
	}

	parts := strings.SplitN(row.Repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", row.Repo)
	}

	client, err := ghpkg.GetGitHubClient()
	if err != nil {
		return fmt.Errorf("GitHub client not available: %w", err)
	}

	commitMsg := fmt.Sprintf("Merge feature PR #%d: %s", row.PRNumber, deref(row.PRTitle, ""))
	_, _, err = client.PullRequests.Merge(
		ctx, parts[0], parts[1], row.PRNumber,
		commitMsg,
		&github.PullRequestOptions{MergeMethod: "squash"},
	)
	if err != nil {
		return fmt.Errorf("failed to merge PR on GitHub: %w", err)
	}

	return nil
}

func (h *Handler) GetStats(ctx context.Context) (*StatsResponse, error) {
	stats, err := h.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return &StatsResponse{
		Pending:      stats[StatusPending],
		PlanComputed: stats[StatusPlanComputed],
		Approved:     stats[StatusApproved],
		Rejected:     stats[StatusRejected],
		Merged:       stats[StatusMerged],
		Closed:       stats[StatusClosed],
	}, nil
}

// --- Internal helpers ---

func (h *Handler) fetchManifest(ctx context.Context, repoFullName, branch string) (string, error) {
	client, err := ghpkg.GetGitHubClient()
	if err != nil {
		return "", fmt.Errorf("GitHub client not available: %w", err)
	}

	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo format: %s", repoFullName)
	}

	content, err := ghpkg.GetFile(ctx, client, parts[1], manifestPath, branch)
	if err != nil {
		return "", err
	}
	decoded, err := content.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode manifest content: %w", err)
	}
	return decoded, nil
}

func (h *Handler) callHorizonPlan(ctx context.Context, manifest Manifest) (interface{}, error) {
	assets := make([]map[string]interface{}, len(manifest.Assets))
	for i, a := range manifest.Assets {
		inputs := make([]map[string]interface{}, len(a.Inputs))
		for j, inp := range a.Inputs {
			m := map[string]interface{}{
				"name":      inp.Name,
				"partition": inp.Partition,
				"type":      inp.Type,
			}
			if inp.WindowSize != nil {
				m["window_size"] = *inp.WindowSize
			}
			inputs[j] = m
		}
		asset := map[string]interface{}{
			"name":        a.Name,
			"version":     a.Version,
			"entity":      a.Entity,
			"entity_key":  a.EntityKey,
			"notebook":    a.Notebook,
			"partition":   a.Partition,
			"trigger":     a.Trigger,
			"serving":     a.Serving,
			"incremental": a.Incremental,
			"inputs":      inputs,
		}
		if a.Schedule != nil {
			asset["schedule"] = *a.Schedule
		}
		if a.Freshness != nil {
			asset["freshness"] = *a.Freshness
		}
		if len(a.Checks) > 0 {
			asset["checks"] = a.Checks
		}
		assets[i] = asset
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"assets":  assets,
		"dry_run": true,
	})
	if err != nil {
		return nil, err
	}

	horizonURL := h.cfg.HorizonInternalURL
	if horizonURL == "" {
		horizonURL = "http://localhost:8080"
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost,
		horizonURL+"/api/v1/fce/plan", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("horizon plan request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read horizon response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("horizon plan returned %d: %s", resp.StatusCode, string(body))
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse horizon plan response: %w", err)
	}
	return result, nil
}

func (h *Handler) approveOnGitHub(repoFullName string, prNumber int, comment string) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return
	}
	client, err := ghpkg.GetGitHubClient()
	if err != nil {
		log.Error().Err(err).Msg("GitHub client not available for PR approval")
		return
	}
	body := "Approved via TruffleBox feature review"
	if comment != "" {
		body = comment
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _, err = client.PullRequests.CreateReview(ctx, parts[0], parts[1], prNumber,
		&github.PullRequestReviewRequest{
			Body:  &body,
			Event: github.String("APPROVE"),
		})
	if err != nil {
		log.Error().Err(err).Int("pr", prNumber).Msg("failed to approve PR on GitHub")
	}
}

func (h *Handler) commentOnGitHub(repoFullName string, prNumber int, body string) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return
	}
	client, err := ghpkg.GetGitHubClient()
	if err != nil {
		log.Error().Err(err).Msg("GitHub client not available for PR comment")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _, err = client.Issues.CreateComment(ctx, parts[0], parts[1], prNumber,
		&github.IssueComment{Body: &body})
	if err != nil {
		log.Error().Err(err).Int("pr", prNumber).Msg("failed to comment on PR")
	}
}

func (h *Handler) triggerApplyFlow(repoFullName string, reviewID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	manifest, err := h.fetchManifest(ctx, repoFullName, "main")
	if err != nil {
		log.Error().Err(err).Int64("reviewID", reviewID).Msg("apply: failed to fetch manifest from main")
		return
	}

	var parsed Manifest
	if err := json.Unmarshal([]byte(manifest), &parsed); err != nil {
		log.Error().Err(err).Msg("apply: failed to parse manifest")
		return
	}

	if err := h.callHorizonRegister(ctx, parsed); err != nil {
		log.Error().Err(err).Msg("apply: horizon register failed")
		return
	}

	if err := h.callHorizonResolve(ctx); err != nil {
		log.Error().Err(err).Msg("apply: horizon resolve failed")
		return
	}

	log.Info().Int64("reviewID", reviewID).Msg("apply flow completed successfully")
}

func (h *Handler) callHorizonRegister(ctx context.Context, manifest Manifest) error {
	assets := make([]map[string]interface{}, len(manifest.Assets))
	for i, a := range manifest.Assets {
		inputs := make([]map[string]interface{}, len(a.Inputs))
		for j, inp := range a.Inputs {
			m := map[string]interface{}{
				"name":      inp.Name,
				"partition": inp.Partition,
				"type":      inp.Type,
			}
			if inp.WindowSize != nil {
				m["window_size"] = *inp.WindowSize
			}
			inputs[j] = m
		}
		asset := map[string]interface{}{
			"name":        a.Name,
			"version":     a.Version,
			"entity":      a.Entity,
			"entity_key":  a.EntityKey,
			"notebook":    a.Notebook,
			"partition":   a.Partition,
			"trigger":     a.Trigger,
			"serving":     a.Serving,
			"incremental": a.Incremental,
			"inputs":      inputs,
		}
		if a.Schedule != nil {
			asset["schedule"] = *a.Schedule
		}
		if a.Freshness != nil {
			asset["freshness"] = *a.Freshness
		}
		if len(a.Checks) > 0 {
			asset["checks"] = a.Checks
		}
		assets[i] = asset
	}

	reqBody, err := json.Marshal(map[string]interface{}{"assets": assets})
	if err != nil {
		return err
	}

	horizonURL := h.cfg.HorizonInternalURL
	if horizonURL == "" {
		horizonURL = "http://localhost:8080"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		horizonURL+"/api/v1/fce/assets/register", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (h *Handler) callHorizonResolve(ctx context.Context) error {
	horizonURL := h.cfg.HorizonInternalURL
	if horizonURL == "" {
		horizonURL = "http://localhost:8080"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		horizonURL+"/api/v1/fce/necessity/resolve", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("resolve request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resolve returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func rowToResponse(row featurereview.ReviewRow) ReviewResponse {
	resp := ReviewResponse{
		ID:        row.ID,
		Repo:      row.Repo,
		PRNumber:  row.PRNumber,
		PRTitle:   deref(row.PRTitle, ""),
		PRAuthor:  deref(row.PRAuthor, ""),
		PRURL:     deref(row.PRURL, ""),
		PRBranch:  deref(row.PRBranch, ""),
		HeadSHA:   deref(row.HeadSHA, ""),
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.ReviewedBy != nil {
		resp.ReviewedBy = *row.ReviewedBy
	}
	resp.ReviewedAt = row.ReviewedAt
	if row.ReviewComment != nil {
		resp.ReviewComment = *row.ReviewComment
	}
	if row.MergedBy != nil {
		resp.MergedBy = *row.MergedBy
	}
	resp.MergedAt = row.MergedAt

	if row.ManifestJSON != nil {
		var manifest Manifest
		if json.Unmarshal([]byte(*row.ManifestJSON), &manifest) == nil {
			resp.AssetSummary = &AssetSummary{
				Total: len(manifest.Assets),
				New:   len(manifest.Assets),
			}
		}
	}
	return resp
}

func deref(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
