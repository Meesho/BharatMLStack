package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/grpc/metadata"
)

const (
	callerIDEnvVar  = "OFS_CALLER_ID"
	authTokenEnvVar = "OFS_AUTH_TOKEN"

	callerIDHeader  = "online-feature-store-caller-id"
	authTokenHeader = "online-feature-store-auth-token"
)

// ToolHandlers holds the feature handler dependency and implements
// all MCP tool handler functions for the online feature store.
type ToolHandlers struct {
	handler retrieve.FeatureServiceServer
}

// NewToolHandlers creates a new ToolHandlers with the given feature service handler.
func NewToolHandlers(handler retrieve.FeatureServiceServer) *ToolHandlers {
	return &ToolHandlers{handler: handler}
}

// RetrieveDecodedFeatures retrieves feature values in human-readable format.
func (t *ToolHandlers) RetrieveDecodedFeatures(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	entityLabel, ok := args["entity_label"].(string)
	if !ok || entityLabel == "" {
		return mcp.NewToolResultError("entity_label is required and must be a string"), nil
	}

	query := &retrieve.Query{
		EntityLabel: entityLabel,
	}

	// Parse keys_schema
	keysSchemaRaw, ok := args["keys_schema"].([]interface{})
	if !ok {
		return mcp.NewToolResultError("keys_schema is required and must be an array of strings"), nil
	}
	for _, v := range keysSchemaRaw {
		s, ok := v.(string)
		if !ok {
			return mcp.NewToolResultError("each element in keys_schema must be a string"), nil
		}
		query.KeysSchema = append(query.KeysSchema, s)
	}

	// Parse feature_groups
	fgRaw, ok := args["feature_groups"].([]interface{})
	if !ok {
		return mcp.NewToolResultError("feature_groups is required and must be an array"), nil
	}
	for _, fgItem := range fgRaw {
		fgMap, ok := fgItem.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("each feature_group must be an object with 'label' and 'feature_labels'"), nil
		}
		label, ok := fgMap["label"].(string)
		if !ok || label == "" {
			return mcp.NewToolResultError("each feature_group must have a 'label' string field"), nil
		}
		labelsRaw, ok := fgMap["feature_labels"].([]interface{})
		if !ok {
			return mcp.NewToolResultError("each feature_group must have a 'feature_labels' array field"), nil
		}
		fg := &retrieve.FeatureGroup{Label: label}
		for _, l := range labelsRaw {
			s, ok := l.(string)
			if !ok {
				return mcp.NewToolResultError("each feature_label must be a string"), nil
			}
			fg.FeatureLabels = append(fg.FeatureLabels, s)
		}
		query.FeatureGroups = append(query.FeatureGroups, fg)
	}

	// Parse keys
	keysRaw, ok := args["keys"].([]interface{})
	if !ok {
		return mcp.NewToolResultError("keys is required and must be an array"), nil
	}
	for _, keyItem := range keysRaw {
		keyArr, ok := keyItem.([]interface{})
		if !ok {
			return mcp.NewToolResultError("each key must be an array of strings"), nil
		}
		key := &retrieve.Keys{}
		for _, k := range keyArr {
			s, ok := k.(string)
			if !ok {
				return mcp.NewToolResultError("each key value must be a string"), nil
			}
			key.Cols = append(key.Cols, s)
		}
		query.Keys = append(query.Keys, key)
	}

	// Inject auth metadata into context
	authCtx := injectAuthContext(ctx)

	result, err := t.handler.RetrieveDecodedResult(authCtx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to retrieve decoded features: %v", err)), nil
	}

	// Build a JSON-friendly response
	type decodedRow struct {
		Keys    []string `json:"keys"`
		Columns []string `json:"columns"`
	}
	type featureInfo struct {
		Label     string `json:"label"`
		ColumnIdx int32  `json:"column_idx"`
	}
	type schemaInfo struct {
		FeatureGroupLabel string        `json:"feature_group_label"`
		Features          []featureInfo `json:"features"`
	}
	type response struct {
		KeysSchema     []string     `json:"keys_schema"`
		FeatureSchemas []schemaInfo `json:"feature_schemas"`
		Rows           []decodedRow `json:"rows"`
	}

	resp := response{
		KeysSchema: result.KeysSchema,
	}
	for _, fs := range result.FeatureSchemas {
		si := schemaInfo{FeatureGroupLabel: fs.FeatureGroupLabel}
		for _, f := range fs.Features {
			si.Features = append(si.Features, featureInfo{
				Label:     f.Label,
				ColumnIdx: f.ColumnIdx,
			})
		}
		resp.FeatureSchemas = append(resp.FeatureSchemas, si)
	}
	for _, row := range result.Rows {
		resp.Rows = append(resp.Rows, decodedRow{
			Keys:    row.Keys,
			Columns: row.Columns,
		})
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// HealthCheck verifies the feature store is operational.
func (t *ToolHandlers) HealthCheck(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`{"status":"ok"}`), nil
}

// injectAuthContext reads OFS_CALLER_ID and OFS_AUTH_TOKEN from env
// and injects them as gRPC incoming metadata on the context.
func injectAuthContext(ctx context.Context) context.Context {
	callerID := os.Getenv(callerIDEnvVar)
	authToken := os.Getenv(authTokenEnvVar)

	if callerID == "" || authToken == "" {
		return ctx
	}

	md := metadata.New(map[string]string{
		callerIDHeader:  callerID,
		authTokenHeader: authToken,
	})
	return metadata.NewIncomingContext(ctx, md)
}
