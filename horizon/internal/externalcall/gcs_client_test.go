package externalcall

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceModelNameInConfig(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		destModelName string
		wantTopLevel  string
		wantNested    string
		expectError   bool
	}{
		{
			name: "replaces top-level name only",
			data: []byte(`
name: "old_model"
instance_group {
  name: "nested_model"
}
`),
			destModelName: "new_model",
			wantTopLevel:  `name: "new_model"`,
			wantNested:    `name: "nested_model"`,
		},
		{
			name:          "single line config",
			data:          []byte(`name: "single_model"` + "\n"),
			destModelName: "replaced_model",
			wantTopLevel:  `name: "replaced_model"`,
		},
		{
			name:          "invalid pbtxt returns error",
			data:          []byte(`invalid_field: "x"`),
			destModelName: "any",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReplaceModelNameInConfig(tt.data, tt.destModelName)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Contains(t, string(got), tt.wantTopLevel)

			if tt.wantNested != "" {
				assert.Contains(t, string(got), tt.wantNested)
			}
		})
	}
}

func TestErrStopIteration(t *testing.T) {
	assert.Error(t, ErrStopIteration)
}

func TestGCSClient_NilClient_ListFolders(t *testing.T) {
	g := &GCSClient{client: nil, ctx: context.Background()}
	folders, err := g.ListFolders("bucket", "prefix/")
	require.Error(t, err)
	assert.Nil(t, folders)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestGCSClient_NilClient_UploadFile(t *testing.T) {
	g := &GCSClient{client: nil, ctx: context.Background()}
	err := g.UploadFile("bucket", "path/obj", []byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestGCSClient_NilClient_CheckFileExists(t *testing.T) {
	g := &GCSClient{client: nil, ctx: context.Background()}
	exists, err := g.CheckFileExists("bucket", "path/obj")
	require.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestGCSClient_NilClient_CheckFolderExists(t *testing.T) {
	g := &GCSClient{client: nil, ctx: context.Background()}
	exists, err := g.CheckFolderExists("bucket", "folder/")
	require.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestGCSClient_NilClient_GetFolderInfo(t *testing.T) {
	g := &GCSClient{client: nil, ctx: context.Background()}
	info, err := g.GetFolderInfo("bucket", "folder/")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestGCSClient_NilClient_ListFoldersWithTimestamp(t *testing.T) {
	g := &GCSClient{client: nil, ctx: context.Background()}
	folders, err := g.ListFoldersWithTimestamp("bucket", "prefix/")
	require.Error(t, err)
	assert.Nil(t, folders)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestGCSClient_NilClient_FindFileWithSuffix(t *testing.T) {
	g := &GCSClient{client: nil, ctx: context.Background()}
	exists, name, err := g.FindFileWithSuffix("bucket", "folder/", ".pbtxt")
	require.Error(t, err)
	assert.False(t, exists)
	assert.Empty(t, name)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestGCSFolderInfo_ZeroValue(t *testing.T) {
	var info GCSFolderInfo
	assert.Empty(t, info.Name)
	assert.Empty(t, info.Path)
	assert.Zero(t, info.FileCount)
	assert.Zero(t, info.Size)
}
