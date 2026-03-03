package featurereview

import (
	"context"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReviewRow struct {
	ID            int64
	Repo          string
	PRNumber      int
	PRTitle       *string
	PRAuthor      *string
	PRURL         *string
	PRBranch      *string
	HeadSHA       *string
	ManifestJSON  *string
	PlanJSON      *string
	Status        string
	ReviewedBy    *string
	ReviewedAt    *time.Time
	ReviewComment *string
	MergedBy      *string
	MergedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Repository interface {
	Upsert(ctx context.Context, row ReviewRow) (*ReviewRow, error)
	GetByID(ctx context.Context, id int64) (*ReviewRow, error)
	GetByRepoPR(ctx context.Context, repo string, prNumber int) (*ReviewRow, error)
	List(ctx context.Context, status, repo string, page, perPage int) ([]ReviewRow, int64, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdatePlan(ctx context.Context, id int64, manifestJSON, planJSON string) error
	UpdateReview(ctx context.Context, id int64, status, reviewedBy, comment string) error
	UpdateMerge(ctx context.Context, id int64, mergedBy string) error
	GetStats(ctx context.Context) (map[string]int64, error)
}

type FeatureReviewRepo struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (Repository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}
	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}
	return &FeatureReviewRepo{
		db: session.(*gorm.DB),
	}, nil
}

func toTable(row ReviewRow) Table {
	return Table{
		Repo:          row.Repo,
		PRNumber:      row.PRNumber,
		PRTitle:       row.PRTitle,
		PRAuthor:      row.PRAuthor,
		PRURL:         row.PRURL,
		PRBranch:      row.PRBranch,
		HeadSHA:       row.HeadSHA,
		ManifestJSON:  row.ManifestJSON,
		PlanJSON:      row.PlanJSON,
		Status:        row.Status,
		ReviewedBy:    row.ReviewedBy,
		ReviewedAt:    row.ReviewedAt,
		ReviewComment: row.ReviewComment,
		MergedBy:      row.MergedBy,
		MergedAt:      row.MergedAt,
	}
}

func toRow(t Table) ReviewRow {
	return ReviewRow{
		ID:            int64(t.ID),
		Repo:          t.Repo,
		PRNumber:      t.PRNumber,
		PRTitle:       t.PRTitle,
		PRAuthor:      t.PRAuthor,
		PRURL:         t.PRURL,
		PRBranch:      t.PRBranch,
		HeadSHA:       t.HeadSHA,
		ManifestJSON:  t.ManifestJSON,
		PlanJSON:      t.PlanJSON,
		Status:        t.Status,
		ReviewedBy:    t.ReviewedBy,
		ReviewedAt:    t.ReviewedAt,
		ReviewComment: t.ReviewComment,
		MergedBy:      t.MergedBy,
		MergedAt:      t.MergedAt,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

func (r *FeatureReviewRepo) Upsert(ctx context.Context, row ReviewRow) (*ReviewRow, error) {
	record := toTable(row)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "repo"}, {Name: "pr_number"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"pr_title", "pr_author", "pr_url", "pr_branch",
			"head_sha", "status",
		}),
	}).Create(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	out := toRow(record)
	return &out, nil
}

func (r *FeatureReviewRepo) GetByID(ctx context.Context, id int64) (*ReviewRow, error) {
	var record Table
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	out := toRow(record)
	return &out, nil
}

func (r *FeatureReviewRepo) GetByRepoPR(ctx context.Context, repo string, prNumber int) (*ReviewRow, error) {
	var record Table
	result := r.db.WithContext(ctx).Where("repo = ? AND pr_number = ?", repo, prNumber).First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	out := toRow(record)
	return &out, nil
}

func (r *FeatureReviewRepo) List(ctx context.Context, status, repo string, page, perPage int) ([]ReviewRow, int64, error) {
	var records []Table
	var total int64

	query := r.db.WithContext(ctx).Model(&Table{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if repo != "" {
		query = query.Where("repo = ?", repo)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]ReviewRow, len(records))
	for i, t := range records {
		rows[i] = toRow(t)
	}
	return rows, total, nil
}

func (r *FeatureReviewRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&Table{}).Where("id = ?", id).
		Update("status", status).Error
}

func (r *FeatureReviewRepo) UpdatePlan(ctx context.Context, id int64, manifestJSON, planJSON string) error {
	return r.db.WithContext(ctx).Model(&Table{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"manifest_json": manifestJSON,
			"plan_json":     planJSON,
			"status":        "plan_computed",
		}).Error
}

func (r *FeatureReviewRepo) UpdateReview(ctx context.Context, id int64, status, reviewedBy, comment string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&Table{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         status,
			"reviewed_by":    reviewedBy,
			"reviewed_at":    &now,
			"review_comment": comment,
		}).Error
}

func (r *FeatureReviewRepo) UpdateMerge(ctx context.Context, id int64, mergedBy string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&Table{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":    "merged",
			"merged_by": mergedBy,
			"merged_at": &now,
		}).Error
}

func (r *FeatureReviewRepo) GetStats(ctx context.Context) (map[string]int64, error) {
	stats := map[string]int64{
		"pending":       0,
		"plan_computed": 0,
		"approved":      0,
		"rejected":      0,
		"merged":        0,
		"closed":        0,
	}
	type statusCount struct {
		Status string
		Count  int64
	}
	var counts []statusCount
	if err := r.db.WithContext(ctx).Model(&Table{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, c := range counts {
		stats[c.Status] = c.Count
	}
	return stats, nil
}
