package mcp

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates and configures a new MCP server with all
// online-feature-store tools registered.
func NewServer(handler retrieve.FeatureServiceServer) *server.MCPServer {
	s := server.NewMCPServer(
		"ofs-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	tools := NewToolHandlers(handler)
	registerTools(s, tools)
	return s
}

func registerTools(s *server.MCPServer, t *ToolHandlers) {
	retrieveDecoded := mcp.NewTool("retrieve_decoded_features",
		mcp.WithDescription("Retrieve feature values in human-readable decoded format for given entity keys across one or more feature groups"),
		mcp.WithString("entity_label",
			mcp.Required(),
			mcp.Description("The entity label to retrieve features for (e.g. 'user', 'catalog')"),
		),
		mcp.WithArray("feature_groups",
			mcp.Required(),
			mcp.Description("Array of feature groups to retrieve. Each element must have 'label' (string) and 'feature_labels' (array of strings)"),
		),
		mcp.WithArray("keys_schema",
			mcp.Required(),
			mcp.Description("Column names for the entity keys (e.g. ['user_id'])"),
		),
		mcp.WithArray("keys",
			mcp.Required(),
			mcp.Description("Array of entity keys. Each element is an array of key values corresponding to keys_schema"),
		),
	)
	s.AddTool(retrieveDecoded, t.RetrieveDecodedFeatures)

	healthCheck := mcp.NewTool("health_check",
		mcp.WithDescription("Check if the online feature store is healthy and reachable"),
	)
	s.AddTool(healthCheck, t.HealthCheck)
}
