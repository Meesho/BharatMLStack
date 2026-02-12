package externalcall

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceModelNameInConfig(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		destModelName  string
		expectContains string
	}{
		{
			name: "replaces top-level name only",
			data: []byte(`name: "old_model"
instance_group {
  name: "old_model"
}
`),
			destModelName:  "new_model",
			expectContains: `name: "new_model"`,
		},
		{
			name: "preserves nested name with indentation",
			data: []byte(`name: "top_level"
  instance_group {
    name: "nested_name"
  }
`),
			destModelName:  "replaced",
			expectContains: `name: "replaced"`,
		},
		{
			name: "single line config",
			data: []byte(`name: "single_model"` + "\n"),
			destModelName:  "replaced_model",
			expectContains: `name: "replaced_model"`,
		},
		{
			name: "no name field returns unchanged",
			data: []byte(`platform: "tensorflow"
version: 1
`),
			destModelName: "any",
			expectContains: `platform: "tensorflow"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceModelNameInConfig(tt.data, tt.destModelName)
			assert.Contains(t, string(got), tt.expectContains)
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
