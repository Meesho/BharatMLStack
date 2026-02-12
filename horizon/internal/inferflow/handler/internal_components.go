package handler

import (
	etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	dbModel "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	mapset "github.com/deckarep/golang-set/v2"
)

// InternalComponentBuilder defines the interface for building internal-only components.
// This interface abstracts all internal feature processing (RTP, SEEN Score, etc.)
// that are only available in internal builds (meesho build tag).
//
// For open-source builds (!meesho), a stub implementation returns empty/pass-through results.
// For internal builds (meesho), the full implementation provides actual functionality.
//
// The external code has NO knowledge of RTP, SEEN Score, or any other internal features.
// All such logic is encapsulated within the internal implementation.
type InternalComponentBuilder interface {
	// IsEnabled returns true if internal components are available in this build
	IsEnabled() bool

	// ProcessFeatures processes the initial feature set and returns additional internal features.
	// It classifies features that are internal-only (RTP, SEEN Score, etc.) and returns:
	// - internalFeatures: features that should be handled by internal components
	// - featureToDataType: data type mappings for internal features
	// The external code will exclude these features from standard processing.
	ProcessFeatures(
		initialFeatures mapset.Set[string],
		featureDataTypes map[string]string,
	) (internalFeatures mapset.Set[string], featureToDataType map[string]string, err error)

	// ClassifyFeature checks if a feature is internal-only.
	// Returns the transformed feature name and true if it's an internal feature type.
	// Returns empty string and false if it's not an internal feature.
	ClassifyFeature(feature string) (transformedFeature string, isInternal bool)

	// GetInternalComponents builds all internal components (RTP, SEEN Score, etc.)
	// Returns the components to be added to the config.
	GetInternalComponents(
		request InferflowOnboardRequest,
		internalFeatures mapset.Set[string],
		etcdConfig etcd.Manager,
		token string,
	) (rtpComponents []RTPComponent, seenScoreComponents []SeenScoreComponent, err error)

	// FetchInternalComponentFeatures fetches features from internal component definitions
	// and classifies them. Returns features that should be added to the main feature set
	// and features that should remain as internal-only.
	FetchInternalComponentFeatures(
		internalFeatures mapset.Set[string],
		etcdConfig etcd.Manager,
	) (fsFeatures mapset.Set[string], newInternalFeatures mapset.Set[string], featureToDataType map[string]string, err error)

	// FetchMissingInternalDataTypes fetches data types for internal features that are missing them
	FetchMissingInternalDataTypes(
		featureToDataType map[string]string,
		internalFeatures mapset.Set[string],
	) error

	// AddInternalDependenciesToDAG adds dependencies for all internal components to the DAG
	AddInternalDependenciesToDAG(
		rtpComponents []RTPComponent,
		seenScoreComponents []SeenScoreComponent,
		featureComponents []FeatureComponent,
		dagConfig *DagExecutionConfig,
	)

	// ============= Adaptor Methods =============

	// AdaptToDBRTPComponent adapts RTP components to DB model format
	AdaptToDBRTPComponent(inferflowConfig InferflowConfig) []dbModel.RTPComponent

	// AdaptToDBSeenScoreComponent adapts SeenScore components to DB model format
	AdaptToDBSeenScoreComponent(inferflowConfig InferflowConfig) []dbModel.SeenScoreComponent

	// AdaptFromDbToRTPComponent adapts DB model RTP components to handler format
	AdaptFromDbToRTPComponent(dbRTPComponents []dbModel.RTPComponent) []RTPComponent

	// AdaptFromDbToSeenScoreComponent adapts DB model SeenScore components to handler format
	AdaptFromDbToSeenScoreComponent(dbSeenScoreComponents []dbModel.SeenScoreComponent) []SeenScoreComponent

	// AdaptToEtcdRTPComponent adapts DB model RTP components to etcd model format
	AdaptToEtcdRTPComponent(dbRTPComponents []dbModel.RTPComponent) []etcd.RTPComponent

	// AdaptToEtcdSeenScoreComponent adapts DB model SeenScore components to etcd model format
	AdaptToEtcdSeenScoreComponent(dbSeenScoreComponents []dbModel.SeenScoreComponent) []etcd.SeenScoreComponent
}

// InternalComponentBuilderInstance is the global instance of the internal component builder.
// This is set by the init() function in either the stub or internal implementation file
// depending on build tags.
var InternalComponentBuilderInstance InternalComponentBuilder
