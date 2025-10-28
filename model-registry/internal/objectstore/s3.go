package objectstore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Client represents an AWS S3 client
type S3Client struct {
	client   *s3.Client
	bucketID string
}

// S3Config represents S3 configuration
type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Endpoint        string // Optional for custom endpoints like MinIO
}

// NewS3Client creates a new S3 client with the provided credentials and bucket ID
func NewS3Client(ctx context.Context, s3Config S3Config, bucketID string) (*S3Client, error) {
	if s3Config.AccessKeyID == "" {
		return nil, fmt.Errorf("access key ID cannot be empty")
	}

	if s3Config.SecretAccessKey == "" {
		return nil, fmt.Errorf("secret access key cannot be empty")
	}

	if s3Config.Region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}

	if bucketID == "" {
		return nil, fmt.Errorf("bucket ID cannot be empty")
	}

	// Create AWS config with static credentials
	var cfg aws.Config
	var err error

	if s3Config.Endpoint != "" {
		// Custom endpoint (e.g., MinIO)
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				s3Config.AccessKeyID,
				s3Config.SecretAccessKey,
				"",
			)),
			config.WithRegion(s3Config.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	} else {
		// Standard AWS S3
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				s3Config.AccessKeyID,
				s3Config.SecretAccessKey,
				"",
			)),
			config.WithRegion(s3Config.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	// Create S3 client
	var client *s3.Client
	if s3Config.Endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s3Config.Endpoint)
			o.UsePathStyle = true // Required for MinIO and some custom endpoints
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	// Validate bucket exists and is accessible
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %w", bucketID, err)
	}

	return &S3Client{
		client:   client,
		bucketID: bucketID,
	}, nil
}

// CreateFolder creates a folder at the given path, creating parent folders if they don't exist
func (s *S3Client) CreateFolder(ctx context.Context, folderPath string) error {
	if folderPath == "" {
		return fmt.Errorf("folder path cannot be empty")
	}

	// Normalize the path - ensure it ends with a slash for folders and starts without slash
	normalizedPath := strings.TrimPrefix(folderPath, "/")
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Get all parent directories that need to be created
	foldersToCreate := s.getParentFolders(normalizedPath)

	// Create folders from root to target
	for _, folder := range foldersToCreate {
		if exists, err := s.folderExists(ctx, folder); err != nil {
			return fmt.Errorf("failed to check if folder %s exists: %w", folder, err)
		} else if exists {
			continue // Skip if folder already exists
		}

		// Create the folder by uploading an empty object with folder path
		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.bucketID),
			Key:         aws.String(folder),
			Body:        strings.NewReader(""),
			ContentType: aws.String("application/x-directory"),
		})
		if err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
	}

	return nil
}

// getParentFolders returns all parent folders that need to be created for the given path
func (s *S3Client) getParentFolders(path string) []string {
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
func (s *S3Client) folderExists(ctx context.Context, folderPath string) (bool, error) {
	// Check if the exact folder object exists
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketID),
		Key:    aws.String(folderPath),
	})
	if err == nil {
		return true, nil
	}

	var notFound *types.NotFound
	if !errors.As(err, &notFound) {
		return false, err
	}

	// Check if any objects exist with this prefix (indicating folder exists with files)
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucketID),
		Prefix:  aws.String(folderPath),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false, err
	}

	return len(result.Contents) > 0, nil
}

// GetHierarchy returns the hierarchical structure of folders and files for the given path
func (s *S3Client) GetHierarchy(ctx context.Context, path string) (*FileInfo, error) {
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

	if err := s.buildHierarchy(ctx, root, normalizedPath); err != nil {
		return nil, fmt.Errorf("failed to build hierarchy for path %s: %w", path, err)
	}

	return root, nil
}

// buildHierarchy recursively builds the file hierarchy for a given directory
func (s *S3Client) buildHierarchy(ctx context.Context, parent *FileInfo, prefix string) error {
	// List all objects with the given prefix
	var allObjects []types.Object

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketID),
		Prefix: aws.String(prefix),
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("error listing objects: %w", err)
		}
		allObjects = append(allObjects, result.Contents...)
	}

	// Build directory structure from all objects
	dirMap := make(map[string]*FileInfo)
	var items []FileInfo

	for _, obj := range allObjects {
		key := aws.ToString(obj.Key)

		// Skip the prefix itself
		if key == prefix {
			continue
		}

		relativePath := strings.TrimPrefix(key, prefix)
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
			if !strings.HasSuffix(key, "/") {
				fileInfo := FileInfo{
					Name:     firstPart,
					Path:     key,
					IsDir:    false,
					Size:     aws.ToInt64(obj.Size),
					Modified: aws.ToTime(obj.LastModified),
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
				if err := s.buildHierarchy(ctx, dirInfo, dirPath); err != nil {
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
func (s *S3Client) ListObjects(ctx context.Context, prefix string) ([]types.Object, error) {
	var allObjects []types.Object

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketID),
		Prefix: aws.String(prefix),
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing objects: %w", err)
		}
		allObjects = append(allObjects, result.Contents...)
	}

	return allObjects, nil
}

// GetBucketName returns the bucket name
func (s *S3Client) GetBucketName() string {
	return s.bucketID
}

// UploadFile uploads a file to S3
func (s *S3Client) UploadFile(ctx context.Context, key string, body *strings.Reader, contentType string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketID),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file %s: %w", key, err)
	}

	return nil
}

// DeleteObject deletes an object from S3
func (s *S3Client) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketID),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}

	return nil
}

// GetObjectURL generates a presigned URL for an object
func (s *S3Client) GetObjectURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key cannot be empty")
	}

	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketID),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s: %w", key, err)
	}

	return request.URL, nil
}
