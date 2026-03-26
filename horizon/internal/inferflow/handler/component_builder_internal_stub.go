//go:build !meesho

package handler

import (
	etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	dbModel "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	mapset "github.com/deckarep/golang-set/v2"
)

// internalComponentBuilderStub is the stub implementation for open-source builds.
// It has NO knowledge of RTP, SEEN Score, or any other internal features.
// All internal processing is a no-op.
type internalComponentBuilderStub struct{}

func init() {
	InternalComponentBuilderInstance = &internalComponentBuilderStub{}
}

// IsEnabled returns false - internal components not available in open-source builds
func (s *internalComponentBuilderStub) IsEnabled() bool {
	return false
}

// ProcessFeatures returns empty results - no internal features in open-source builds
func (s *internalComponentBuilderStub) ProcessFeatures(
	initialFeatures mapset.Set[string],
	featureDataTypes map[string]string,
) (mapset.Set[string], map[string]string, error) {
	return mapset.NewSet[string](), make(map[string]string), nil
}

// ClassifyFeature returns false - no features are internal in open-source builds
func (s *internalComponentBuilderStub) ClassifyFeature(feature string) (string, bool) {
	return "", false
}

// GetInternalComponents returns empty slices - no internal components in open-source builds
func (s *internalComponentBuilderStub) GetInternalComponents(
	request InferflowOnboardRequest,
	internalFeatures mapset.Set[string],
	etcdConfig etcd.Manager,
	token string,
) ([]RTPComponent, []SeenScoreComponent, error) {
	return []RTPComponent{}, []SeenScoreComponent{}, nil
}

// FetchInternalComponentFeatures returns empty results - no internal components in open-source builds
func (s *internalComponentBuilderStub) FetchInternalComponentFeatures(
	internalFeatures mapset.Set[string],
	etcdConfig etcd.Manager,
) (mapset.Set[string], mapset.Set[string], map[string]string, error) {
	return mapset.NewSet[string](), mapset.NewSet[string](), make(map[string]string), nil
}

// FetchMissingInternalDataTypes is a no-op - no internal features in open-source builds
func (s *internalComponentBuilderStub) FetchMissingInternalDataTypes(
	featureToDataType map[string]string,
	internalFeatures mapset.Set[string],
) error {
	return nil
}

// AddInternalDependenciesToDAG is a no-op - no internal components in open-source builds
func (s *internalComponentBuilderStub) AddInternalDependenciesToDAG(
	rtpComponents []RTPComponent,
	seenScoreComponents []SeenScoreComponent,
	featureComponents []FeatureComponent,
	dagConfig *DagExecutionConfig,
) {
	// No-op for open-source builds
}

// ============= Adaptor Stub Methods =============

// AdaptToDBRTPComponent returns empty slice - no RTP components in open-source builds
func (s *internalComponentBuilderStub) AdaptToDBRTPComponent(inferflowConfig InferflowConfig) []dbModel.RTPComponent {
	return []dbModel.RTPComponent{}
}

// AdaptToDBSeenScoreComponent returns empty slice - no SeenScore components in open-source builds
func (s *internalComponentBuilderStub) AdaptToDBSeenScoreComponent(inferflowConfig InferflowConfig) []dbModel.SeenScoreComponent {
	return []dbModel.SeenScoreComponent{}
}

// AdaptFromDbToRTPComponent returns empty slice - no RTP components in open-source builds
func (s *internalComponentBuilderStub) AdaptFromDbToRTPComponent(dbRTPComponents []dbModel.RTPComponent) []RTPComponent {
	return []RTPComponent{}
}

// AdaptFromDbToSeenScoreComponent returns empty slice - no SeenScore components in open-source builds
func (s *internalComponentBuilderStub) AdaptFromDbToSeenScoreComponent(dbSeenScoreComponents []dbModel.SeenScoreComponent) []SeenScoreComponent {
	return []SeenScoreComponent{}
}

// AdaptToEtcdRTPComponent returns empty slice - no RTP components in open-source builds
func (s *internalComponentBuilderStub) AdaptToEtcdRTPComponent(dbRTPComponents []dbModel.RTPComponent) []etcd.RTPComponent {
	return []etcd.RTPComponent{}
}

// AdaptToEtcdSeenScoreComponent returns empty slice - no SeenScore components in open-source builds
func (s *internalComponentBuilderStub) AdaptToEtcdSeenScoreComponent(dbSeenScoreComponents []dbModel.SeenScoreComponent) []etcd.SeenScoreComponent {
	return []etcd.SeenScoreComponent{}
}
