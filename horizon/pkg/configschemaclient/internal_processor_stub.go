//go:build !meesho

package configschemaclient

// internalSchemaProcessorStub is the stub implementation for open-source builds.
// It returns empty results for all internal component processing.
type internalSchemaProcessorStub struct{}

func init() {
	InternalSchemaProcessorInstance = &internalSchemaProcessorStub{}
}

// ProcessRTP returns empty slice - no RTP processing in open-source builds
func (s *internalSchemaProcessorStub) ProcessRTP(rtpComponents []RTPComponent) []SchemaComponents {
	return nil
}

// ProcessSeenScore returns empty slice - no SeenScore processing in open-source builds
func (s *internalSchemaProcessorStub) ProcessSeenScore(seenScoreComponents []SeenScoreComponent) []SchemaComponents {
	return nil
}
