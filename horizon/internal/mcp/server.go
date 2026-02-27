package mcp

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/handler"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates and configures a new MCP server with all
// Horizon online-feature-store discovery tools registered.
func NewServer(configHandler handler.Config) *server.MCPServer {
	s := server.NewMCPServer(
		"horizon-ofs-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	tools := NewToolHandlers(configHandler)
	registerTools(s, tools)
	return s
}

func registerTools(s *server.MCPServer, t *ToolHandlers) {
	listEntities := mcp.NewTool("list_entities",
		mcp.WithDescription("List all registered entity labels in the online feature store"),
	)
	s.AddTool(listEntities, t.ListEntities)

	getEntityDetails := mcp.NewTool("get_entity_details",
		mcp.WithDescription("Get detailed configuration for all entities including keys and cache settings"),
	)
	s.AddTool(getEntityDetails, t.GetEntityDetails)

	listFeatureGroups := mcp.NewTool("list_feature_groups",
		mcp.WithDescription("List feature groups for a given entity with their labels and data types"),
		mcp.WithString("entity_label",
			mcp.Required(),
			mcp.Description("The entity label to list feature groups for"),
		),
	)
	s.AddTool(listFeatureGroups, t.ListFeatureGroups)

	getFeatureGroupDetails := mcp.NewTool("get_feature_group_details",
		mcp.WithDescription("Get detailed configuration for feature groups including features, versions, TTL, and cache settings"),
		mcp.WithString("entity_label",
			mcp.Required(),
			mcp.Description("The entity label to retrieve feature group details for"),
		),
	)
	s.AddTool(getFeatureGroupDetails, t.GetFeatureGroupDetails)
}
