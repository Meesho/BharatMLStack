package route

import (
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/featurereview/controller"
	"github.com/Meesho/BharatMLStack/horizon/internal/featurereview/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/featurereview"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var initRouterOnce sync.Once

func Init() {
	initRouterOnce.Do(func() {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("FeatureReview: failed to get SQL connection")
		}
		sqlConn, ok := conn.(*infra.SQLConnection)
		if !ok {
			log.Fatal().Msg("FeatureReview: SQL connection is not of type *infra.SQLConnection")
		}

		repo, err := featurereview.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("FeatureReview: failed to create repository")
		}

		reposStr := viper.GetString("fce_feature_repos")
		var featureRepos []string
		if reposStr != "" {
			for _, r := range strings.Split(reposStr, ",") {
				trimmed := strings.TrimSpace(r)
				if trimmed != "" {
					featureRepos = append(featureRepos, trimmed)
				}
			}
		}

		cfg := handler.Config{
			WebhookSecret:      viper.GetString("fce_webhook_secret"),
			FeatureRepos:       featureRepos,
			HorizonInternalURL: viper.GetString("fce_horizon_internal_url"),
		}

		h := handler.New(cfg, repo)
		ctrl := controller.New(h)

		webhooks := httpframework.Instance().Group("/webhooks")
		{
			webhooks.POST("/github", ctrl.HandleWebhook)
		}

		api := httpframework.Instance().Group("/api/v1/feature-reviews")
		{
			api.GET("", ctrl.ListReviews)
			api.GET("/stats", ctrl.GetStats)
			api.GET("/:id", ctrl.GetReview)
			api.POST("/:id/compute-plan", ctrl.ComputePlan)
			api.POST("/:id/approve", ctrl.Approve)
			api.POST("/:id/reject", ctrl.Reject)
			api.POST("/:id/merge", ctrl.MergePR)
		}

		log.Info().Int("watched_repos", len(featureRepos)).Msg("FeatureReview routes registered")
	})
}
