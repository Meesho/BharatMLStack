package featurereview

import (
	"time"

	"gorm.io/gorm"
)

const (
	tableName = "feature_reviews"
	createdAt = "CreatedAt"
	updatedAt = "UpdatedAt"
)

type Table struct {
	ID            uint       `gorm:"primaryKey;autoIncrement;column:id"`
	Repo          string     `gorm:"column:repo;not null;size:255;uniqueIndex:idx_fr_repo_pr"`
	PRNumber      int        `gorm:"column:pr_number;not null;uniqueIndex:idx_fr_repo_pr"`
	PRTitle       *string    `gorm:"column:pr_title;size:500"`
	PRAuthor      *string    `gorm:"column:pr_author;size:255"`
	PRURL         *string    `gorm:"column:pr_url;size:500"`
	PRBranch      *string    `gorm:"column:pr_branch;size:255"`
	HeadSHA       *string    `gorm:"column:head_sha;size:64"`
	ManifestJSON  *string    `gorm:"column:manifest_json;type:text"`
	PlanJSON      *string    `gorm:"column:plan_json;type:text"`
	Status        string     `gorm:"column:status;not null;size:50;default:pending;index:idx_fr_status"`
	ReviewedBy    *string    `gorm:"column:reviewed_by;size:255"`
	ReviewedAt    *time.Time `gorm:"column:reviewed_at"`
	ReviewComment *string    `gorm:"column:review_comment;type:text"`
	MergedBy      *string    `gorm:"column:merged_by;size:255"`
	MergedAt      *time.Time `gorm:"column:merged_at"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Table) TableName() string {
	return tableName
}

func (Table) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(createdAt, time.Now())
	return
}

func (Table) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.SetColumn(updatedAt, time.Now())
	return
}
