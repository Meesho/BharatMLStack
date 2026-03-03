package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/featurecomputeengine/controller"
	"github.com/Meesho/BharatMLStack/horizon/internal/featurecomputeengine/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetdeps"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetregistry"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetstate"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/datasetpartitions"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/materializations"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

var initRouterOnce sync.Once

// Init creates all FCE dependencies (repositories, handler, controller)
// and registers HTTP routes. Must be called after infra.InitDBConnectors
// and httpframework.Init.
func Init() {
	initRouterOnce.Do(func() {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("FCE: failed to get SQL connection")
		}
		sqlConn, ok := conn.(*infra.SQLConnection)
		if !ok {
			log.Fatal().Msg("FCE: SQL connection is not of type *infra.SQLConnection")
		}

		ar, err := assetregistry.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("FCE: failed to create asset registry repository")
		}
		ad, err := assetdeps.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("FCE: failed to create asset deps repository")
		}
		as, err := assetstate.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("FCE: failed to create asset state repository")
		}
		mat, err := materializations.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("FCE: failed to create materializations repository")
		}
		dp, err := datasetpartitions.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("FCE: failed to create dataset partitions repository")
		}

		cfg := handler.Config{
			ArtifactBasePath: "/_artifacts",
			ServingBasePath:  "/serving",
			DefaultPartition: "ds",
		}

		h := handler.New(cfg, ar, ad, as, mat, dp)
		ctrl := controller.New(h)

		fce := httpframework.Instance().Group("/api/v1/fce")
		{
			fce.POST("/assets/register", ctrl.RegisterAssets)

			fce.POST("/execution-plan", ctrl.GetExecutionPlan)

			fce.POST("/assets/ready", ctrl.ReportAssetReady)
			fce.POST("/assets/failed", ctrl.ReportAssetFailed)

			fce.GET("/necessity", ctrl.GetAllNecessity)
			fce.GET("/necessity/:asset_name", ctrl.GetNecessity)
			fce.POST("/serving/override", ctrl.SetServingOverride)
			fce.DELETE("/serving/override/:asset_name", ctrl.ClearServingOverride)
			fce.POST("/necessity/resolve", ctrl.ResolveNecessity)

			fce.GET("/lineage", ctrl.GetFullLineage)
			fce.GET("/lineage/:asset_name/upstream", ctrl.GetUpstream)
			fce.GET("/lineage/:asset_name/downstream", ctrl.GetDownstream)

			fce.POST("/plan", ctrl.ComputePlan)
			fce.POST("/plan/apply", ctrl.ApplyPlan)

			fce.POST("/datasets/ready", ctrl.MarkDatasetReady)
			fce.GET("/datasets/:dataset_name/partitions", ctrl.GetDatasetPartitions)
		}
	})
}
