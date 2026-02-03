package configschemaclient

// InternalSchemaProcessor defines the interface for processing internal-only components
// (RTP, SeenScore) in the schema builder.
//
// For open-source builds (!meesho), a stub implementation returns empty results.
// For internal builds (meesho), the full implementation provides actual processing.
type InternalSchemaProcessor interface {
	// ProcessRTP processes RTP components and returns schema components
	ProcessRTP(rtpComponents []RTPComponent) []SchemaComponents

	// ProcessSeenScore processes SeenScore components and returns schema components
	ProcessSeenScore(seenScoreComponents []SeenScoreComponent) []SchemaComponents
}

// InternalSchemaProcessorInstance is the global instance of the internal schema processor.
// This is set by the init() function in either the stub or internal implementation file
// depending on build tags.
var InternalSchemaProcessorInstance InternalSchemaProcessor
