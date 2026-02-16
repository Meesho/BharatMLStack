package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/handler"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandlers holds the handler.Config dependency and implements
// all MCP tool handler functions for Horizon discovery tools.
type ToolHandlers struct {
	config handler.Config
}

// NewToolHandlers creates a new ToolHandlers with the given config handler.
func NewToolHandlers(config handler.Config) *ToolHandlers {
	return &ToolHandlers{config: config}
}

// ListEntities returns all registered entity labels.
func (t *ToolHandlers) ListEntities(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entities, err := t.config.GetAllEntities()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list entities: %v", err)), nil
	}

	data, err := json.Marshal(entities)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal entities: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// GetEntityDetails returns full configuration for all entities
// including keys, in-memory cache, and distributed cache settings.
func (t *ToolHandlers) GetEntityDetails(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entities, err := t.config.RetrieveEntities()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to retrieve entity details: %v", err)), nil
	}

	data, err := json.Marshal(entities)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal entity details: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// ListFeatureGroups returns feature groups for a given entity
// with their labels and data types.
func (t *ToolHandlers) ListFeatureGroups(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entityLabel, err := request.RequireString("entity_label")
	if err != nil {
		return mcp.NewToolResultError("entity_label is required"), nil
	}

	featureGroups, err := t.config.RetrieveFeatureGroups(entityLabel)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list feature groups: %v", err)), nil
	}

	type fgSummary struct {
		Label    string `json:"label"`
		DataType string `json:"data-type"`
	}
	var summaries []fgSummary
	if featureGroups != nil {
		for _, fg := range *featureGroups {
			summaries = append(summaries, fgSummary{
				Label:    fg.FeatureGroupLabel,
				DataType: string(fg.DataType),
			})
		}
	}

	data, err := json.Marshal(summaries)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal feature groups: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// GetFeatureGroupDetails returns full feature group configuration
// including features, active version, TTL, and cache settings.
func (t *ToolHandlers) GetFeatureGroupDetails(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entityLabel, err := request.RequireString("entity_label")
	if err != nil {
		return mcp.NewToolResultError("entity_label is required"), nil
	}

	featureGroups, err := t.config.RetrieveFeatureGroups(entityLabel)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to retrieve feature group details: %v", err)), nil
	}

	data, err := json.Marshal(featureGroups)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal feature group details: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
