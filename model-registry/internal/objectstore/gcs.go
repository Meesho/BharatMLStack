package objectstore

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSClient represents a Google Cloud Storage client
type GCSClient struct {
	client   *storage.Client
	bucketID string
	bucket   *storage.BucketHandle
}

// FileInfo represents information about a file or folder in GCS
type FileInfo struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"is_dir"`
	Size     int64      `json:"size,omitempty"`
	Modified time.Time  `json:"modified,omitempty"`
	Children []FileInfo `json:"children,omitempty"`
}

// NewGCSClient creates a new GCS client with the provided credentials and bucket ID
func NewGCSClient(ctx context.Context, credentialsJSON string, bucketID string) (*GCSClient, error) {
	if credentialsJSON == "" {
		return nil, fmt.Errorf("credentials JSON string cannot be empty")
	}

	if bucketID == "" {
		return nil, fmt.Errorf("bucket ID cannot be empty")
	}

	// Validate credentials JSON format
	var creds map[string]interface{}
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return nil, fmt.Errorf("invalid credentials JSON format: %w", err)
	}

	// Create storage client with credentials
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(credentialsJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Validate bucket exists and is accessible
	bucket := client.Bucket(bucketID)
	if _, err := bucket.Attrs(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to access bucket %s: %w", bucketID, err)
	}

	return &GCSClient{
		client:   client,
		bucketID: bucketID,
		bucket:   bucket,
	}, nil
}

// Close closes the GCS client
func (g *GCSClient) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// CreateFolder creates a folder at the given path, creating parent folders if they don't exist
func (g *GCSClient) CreateFolder(ctx context.Context, folderPath string) error {
	if folderPath == "" {
		return fmt.Errorf("folder path cannot be empty")
	}

	// Normalize the path - ensure it ends with a slash for folders and starts without slash
	normalizedPath := strings.TrimPrefix(folderPath, "/")
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Get all parent directories that need to be created
	foldersToCreate := g.getParentFolders(normalizedPath)

	// Create folders from root to target
	for _, folder := range foldersToCreate {
		if exists, err := g.folderExists(ctx, folder); err != nil {
			return fmt.Errorf("failed to check if folder %s exists: %w", folder, err)
		} else if exists {
			continue // Skip if folder already exists
		}

		// Create the folder by uploading an empty object with folder path
		obj := g.bucket.Object(folder)
		writer := obj.NewWriter(ctx)
		writer.ContentType = "application/x-directory"

		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
	}

	return nil
}

// getParentFolders returns all parent folders that need to be created for the given path
func (g *GCSClient) getParentFolders(path string) []string {
	var folders []string
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")

	currentPath := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		currentPath += part + "/"
		folders = append(folders, currentPath)
	}

	return folders
}

// folderExists checks if a folder exists in the bucket
func (g *GCSClient) folderExists(ctx context.Context, folderPath string) (bool, error) {
	// Check if the exact folder object exists
	obj := g.bucket.Object(folderPath)
	if _, err := obj.Attrs(ctx); err == nil {
		return true, nil
	} else if err != storage.ErrObjectNotExist {
		return false, err
	}

	// Check if any objects exist with this prefix (indicating folder exists with files)
	query := &storage.Query{
		Prefix:    folderPath,
		Delimiter: "/",
	}

	it := g.bucket.Objects(ctx, query)
	_, err := it.Next()
	if err == nil {
		return true, nil // Found at least one object with this prefix
	}
	if err == storage.ErrObjectNotExist {
		return false, nil
	}

	return false, err
}

// GetHierarchy returns the hierarchical structure of folders and files for the given path
func (g *GCSClient) GetHierarchy(ctx context.Context, path string) (*FileInfo, error) {
	// Normalize the path
	normalizedPath := strings.TrimPrefix(path, "/")
	if normalizedPath != "" && !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Build the file hierarchy
	root := &FileInfo{
		Name:     filepath.Base(strings.TrimSuffix(normalizedPath, "/")),
		Path:     normalizedPath,
		IsDir:    true,
		Children: []FileInfo{},
	}

	if root.Name == "." || root.Name == "" {
		root.Name = "/"
		root.Path = ""
	}

	if err := g.buildHierarchy(ctx, root, normalizedPath); err != nil {
		return nil, fmt.Errorf("failed to build hierarchy for path %s: %w", path, err)
	}

	return root, nil
}

// buildHierarchy recursively builds the file hierarchy for a given directory
func (g *GCSClient) buildHierarchy(ctx context.Context, parent *FileInfo, prefix string) error {
	// Use a simpler approach: list all objects without delimiter first to find all directories
	allObjectsQuery := &storage.Query{Prefix: prefix}
	allIt := g.bucket.Objects(ctx, allObjectsQuery)

	var allObjects []*storage.ObjectAttrs
	for {
		attrs, err := allIt.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing all objects: %w", err)
		}
		allObjects = append(allObjects, attrs)
	}

	// Build directory structure from all objects
	dirMap := make(map[string]*FileInfo)
	var items []FileInfo

	for _, attrs := range allObjects {
		// Skip the prefix itself
		if attrs.Name == prefix {
			continue
		}

		relativePath := strings.TrimPrefix(attrs.Name, prefix)
		if relativePath == "" {
			continue
		}

		// Split path into parts
		parts := strings.Split(strings.TrimSuffix(relativePath, "/"), "/")
		if len(parts) == 0 {
			continue
		}

		// Handle direct children only (first level)
		firstPart := parts[0]
		if len(parts) == 1 {
			// This is a direct file
			if !strings.HasSuffix(attrs.Name, "/") {
				fileInfo := FileInfo{
					Name:     firstPart,
					Path:     attrs.Name,
					IsDir:    false,
					Size:     attrs.Size,
					Modified: attrs.Updated,
				}
				items = append(items, fileInfo)
			}
		} else {
			// This is inside a subdirectory
			dirPath := prefix + firstPart + "/"
			if _, exists := dirMap[dirPath]; !exists {
				dirInfo := &FileInfo{
					Name:     firstPart,
					Path:     dirPath,
					IsDir:    true,
					Children: []FileInfo{},
				}
				dirMap[dirPath] = dirInfo

				// Recursively build this directory
				if err := g.buildHierarchy(ctx, dirInfo, dirPath); err != nil {
					return err
				}

				items = append(items, *dirInfo)
			}
		}
	}

	// Sort items: directories first, then files, both alphabetically
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir && !items[j].IsDir {
			return true
		}
		if !items[i].IsDir && items[j].IsDir {
			return false
		}
		return items[i].Name < items[j].Name
	})

	parent.Children = items
	return nil
}

// ListObjects lists all objects in the bucket with optional prefix filtering
func (g *GCSClient) ListObjects(ctx context.Context, prefix string) ([]*storage.ObjectAttrs, error) {
	query := &storage.Query{Prefix: prefix}
	it := g.bucket.Objects(ctx, query)

	var objects []*storage.ObjectAttrs
	for {
		attrs, err := it.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing objects: %w", err)
		}
		objects = append(objects, attrs)
	}

	return objects, nil
}

// GetBucketName returns the bucket name
func (g *GCSClient) GetBucketName() string {
	return g.bucketID
}
