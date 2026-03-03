package handler

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/featurereview"
)

type Handler struct {
	cfg  Config
	repo featurereview.Repository
}

func New(cfg Config, repo featurereview.Repository) *Handler {
	return &Handler{
		cfg:  cfg,
		repo: repo,
	}
}
