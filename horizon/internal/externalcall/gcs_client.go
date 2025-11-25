package externalcall

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
)

type GCSClientInterface interface {
	ReadFile(bucket, objectPath string) ([]byte, error)
	TransferFolder(srcBucket, srcPath, srcModelName, destBucket, destPath, destModelName string) error
	TransferAndDeleteFolder(srcBucket, srcPath, srcModelName, destBucket, destPath, destModelName string) error
	DeleteFolder(bucket, modelPath, modelName string) error
	ListFolders(bucket, prefix string) ([]string, error)
	UploadFile(bucket, objectPath string, data []byte) error
	CheckFileExists(bucket, objectPath string) (bool, error)
	CheckFolderExists(bucket, folderPrefix string) (bool, error)
	UploadFolderFromLocal(srcFolderPath, bucket, destPath string) error
	GetFolderInfo(bucket, folderPrefix string) (*GCSFolderInfo, error)
	ListFoldersWithTimestamp(bucket, prefix string) ([]GCSFolderInfo, error)
	FindFileWithSuffix(bucket, folderPath, suffix string) (bool, string, error)
}

const (
	maxRetries         = 3
	initialBackoffTime = 2 * time.Second
	maxConcurrentFiles = 10 // Maximum number of files to transfer concurrently
)

type GCSFolderInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
	Size      int64     `json:"size"`
	FileCount int       `json:"file_count"`
}

type GCSClient struct {
	client *storage.Client
	ctx    context.Context
}

func CreateGCSClient(isGcsEnabled bool) GCSClientInterface {
	if !isGcsEnabled {
		log.Warn().Msg("GCS client is disabled")
		return &GCSClient{
			client: nil,
			ctx:    nil,
		}
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create GCS client")
		return &GCSClient{
			client: nil,
			ctx:    ctx,
		}
	}
	return &GCSClient{
		client: client,
		ctx:    ctx,
	}
}

func (g *GCSClient) ReadFile(bucket, objectPath string) ([]byte, error) {
	rc, err := g.client.Bucket(bucket).Object(objectPath).NewReader(g.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create object reader: %w", err)
	}
	defer rc.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, rc); err != nil {
		return nil, fmt.Errorf("failed to read object into buffer: %w", err)
	}
	return buf.Bytes(), nil
}

func (g *GCSClient) TransferFolder(srcBucket, srcPath, srcModelName, destBucket, destPath, destModelName string) error {
	prefix := path.Join(srcPath, srcModelName)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Phase 1: Collect all objects and separate config files from regular files
	var regularFiles []storage.ObjectAttrs
	var configFiles []storage.ObjectAttrs

	it := g.client.Bucket(srcBucket).Objects(g.ctx, &storage.Query{Prefix: prefix})
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list source bucket: %w", err)
		}

		if strings.HasSuffix(objAttrs.Name, "/") {
			continue
		}

		if strings.HasSuffix(objAttrs.Name, "config.pbtxt") {
			configFiles = append(configFiles, *objAttrs)
		} else {
			regularFiles = append(regularFiles, *objAttrs)
		}
	}

	if len(regularFiles) == 0 && len(configFiles) == 0 {
		log.Info().Msg("No objects found to transfer")
		return nil
	}

	log.Info().Msgf("Starting two-phase transfer: %d regular files, %d config files", len(regularFiles), len(configFiles))

	// Phase 1: Fast direct transfer of regular files (no local download)
	if len(regularFiles) > 0 {
		if err := g.transferRegularFiles(regularFiles, srcBucket, destBucket, destPath, destModelName, prefix); err != nil {
			return fmt.Errorf("failed to transfer regular files: %w", err)
		}
	}

	// Phase 2: Handle config files with model name replacement
	if len(configFiles) > 0 {
		if err := g.transferConfigFiles(configFiles, srcBucket, destBucket, destPath, destModelName, prefix); err != nil {
			return fmt.Errorf("failed to transfer config files: %w", err)
		}
	}

	log.Info().Msgf("Successfully completed two-phase transfer")
	return nil
}

// transferRegularFiles performs fast direct GCS-to-GCS transfers for regular files
func (g *GCSClient) transferRegularFiles(files []storage.ObjectAttrs, srcBucket, destBucket, destPath, destModelName, prefix string) error {
	log.Info().Msgf("Phase 1: Starting fast direct transfer of %d regular files", len(files))

	// Use worker pool for concurrent direct transfers
	semaphore := make(chan struct{}, maxConcurrentFiles)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var transferErrors []error

	for _, objAttrs := range files {
		wg.Add(1)
		go func(obj storage.ObjectAttrs) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			if err := g.transferSingleRegularFile(obj, srcBucket, destBucket, destPath, destModelName, prefix); err != nil {
				mu.Lock()
				transferErrors = append(transferErrors, fmt.Errorf("failed to transfer %s: %w", obj.Name, err))
				mu.Unlock()
			}
		}(objAttrs)
	}

	wg.Wait()

	if len(transferErrors) > 0 {
		return fmt.Errorf("regular file transfer completed with %d errors: %v", len(transferErrors), transferErrors[0])
	}

	log.Info().Msgf("Phase 1 completed: Successfully transferred %d regular files", len(files))
	return nil
}

// transferSingleRegularFile performs direct GCS-to-GCS copy for a single regular file
func (g *GCSClient) transferSingleRegularFile(objAttrs storage.ObjectAttrs, srcBucket, destBucket, destPath, destModelName, prefix string) error {
	// Calculate destination path
	relPath := strings.TrimPrefix(objAttrs.Name, prefix)
	relPath = strings.TrimPrefix(relPath, "/")
	destObjectPath := path.Join(destPath, destModelName, relPath)

	// Use direct GCS-to-GCS copy operation
	srcObj := g.client.Bucket(srcBucket).Object(objAttrs.Name)
	destObj := g.client.Bucket(destBucket).Object(destObjectPath)

	_, err := destObj.CopierFrom(srcObj).Run(g.ctx)
	if err != nil {
		return fmt.Errorf("failed to copy object directly: %w", err)
	}

	log.Debug().Msgf("Direct transfer completed: %s -> %s", objAttrs.Name, destObjectPath)
	return nil
}

// transferConfigFiles handles config.pbtxt files with model name replacement
func (g *GCSClient) transferConfigFiles(files []storage.ObjectAttrs, srcBucket, destBucket, destPath, destModelName, prefix string) error {
	log.Info().Msgf("Phase 2: Processing %d config files with model name replacement", len(files))

	for _, objAttrs := range files {
		if err := g.transferSingleConfigFile(objAttrs, srcBucket, destBucket, destPath, destModelName, prefix); err != nil {
			return fmt.Errorf("failed to transfer config file %s: %w", objAttrs.Name, err)
		}
	}

	log.Info().Msgf("Phase 2 completed: Successfully processed %d config files", len(files))
	return nil
}

// transferSingleConfigFile handles a single config.pbtxt file with model name replacement
func (g *GCSClient) transferSingleConfigFile(objAttrs storage.ObjectAttrs, srcBucket, destBucket, destPath, destModelName, prefix string) error {
	// Calculate destination path
	relPath := strings.TrimPrefix(objAttrs.Name, prefix)
	relPath = strings.TrimPrefix(relPath, "/")
	destObjectPath := path.Join(destPath, destModelName, relPath)

	// Download config file content
	srcReader, err := g.client.Bucket(srcBucket).Object(objAttrs.Name).NewReader(g.ctx)
	if err != nil {
		return fmt.Errorf("failed to create source reader for config: %w", err)
	}
	defer srcReader.Close()

	// Read config content (config files are typically small)
	content, err := io.ReadAll(srcReader)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Replace model name
	log.Info().Msgf("Processing config.pbtxt file: %s -> %s", objAttrs.Name, destObjectPath)
	modified := replaceModelNameInConfig(content, destModelName)

	// Upload modified content
	destWriter := g.client.Bucket(destBucket).Object(destObjectPath).NewWriter(g.ctx)
	defer destWriter.Close()

	_, err = destWriter.Write(modified)
	if err != nil {
		return fmt.Errorf("failed to write modified config: %w", err)
	}

	log.Debug().Msgf("Config file processed: %s -> %s", objAttrs.Name, destObjectPath)
	return nil
}

func (g *GCSClient) DeleteFolder(bucket, modelPath, modelName string) error {
	// Ensure the prefix ends with "/" to avoid matching partial directory names
	prefix := path.Join(modelPath, modelName)

	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	log.Debug().Msgf("Deleting objects with prefix: %s", prefix)
	it := g.client.Bucket(bucket).Objects(g.ctx, &storage.Query{Prefix: prefix})

	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list objects for deletion: %w", err)
		}

		err = g.client.Bucket(bucket).Object(objAttrs.Name).Delete(g.ctx)
		if err != nil {
			log.Printf("failed to delete object %s: %v", objAttrs.Name, err)
			continue
		}
		log.Debug().Msgf("deleted object: %s", objAttrs.Name)
	}
	return nil
}

func (g *GCSClient) TransferAndDeleteFolder(srcBucket, srcPath, srcModelName, destBucket, destPath, destModelName string) error {
	err := g.TransferFolder(srcBucket, srcPath, srcModelName, destBucket, destPath, destModelName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to transfer folder")
		return fmt.Errorf("failed to transfer folder: %w", err)
	}

	backoffTime := initialBackoffTime
	for i := 0; i < maxRetries; i++ {
		err = g.DeleteFolder(srcBucket, srcPath, srcModelName)
		if err == nil {
			break
		}
		log.Error().Err(err).Msgf("Attempt %d: Failed to delete source folder, retrying...", i+1)
		time.Sleep(backoffTime)
		backoffTime *= 2
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to delete source folder after retries, attempting rollback")
		rollbackErr := g.TransferFolder(destBucket, destPath, destModelName, srcBucket, srcPath, srcModelName)
		if rollbackErr != nil {
			log.Error().Err(rollbackErr).Msg("Rollback failed")
			return fmt.Errorf("failed to rollback the transfer after deletion failure: %w", rollbackErr)
		}
		log.Info().Msg("Rollback successful, but deletion still failed")
		return fmt.Errorf("failed to delete source folder after successful transfer: %w", err)
	}
	log.Info().Msg("Successfully transferred and deleted folder")
	return nil
}

// replaceModelNameInConfig modifies the `name:` field in config.pbtxt content
func replaceModelNameInConfig(data []byte, destModelName string) []byte {
	lines := strings.Split(string(data), "\n")
	originalName := ""
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			originalName = line
			lines[i] = fmt.Sprintf(`name: "%s"`, destModelName)
			log.Info().Msgf("Replacing model name in config.pbtxt: '%s' -> 'name: \"%s\"'", originalName, destModelName)
			break
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (g *GCSClient) ListFolders(bucket, prefix string) ([]string, error) {
	if g.client == nil {
		return nil, fmt.Errorf("GCS client not initialized properly")
	}

	var folders []string
	seenFolders := make(map[string]bool)

	// Ensure prefix ends with a trailing slash
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	log.Info().Msgf("Listing folders in GCS bucket %s with prefix %s", bucket, prefix)

	it := g.client.Bucket(bucket).Objects(g.ctx, &storage.Query{
		Prefix: prefix,
		// Do NOT set Delimiter here
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Extract folder name after the prefix
		if attrs.Name != "" {
			trimmed := strings.TrimPrefix(attrs.Name, prefix)
			parts := strings.SplitN(trimmed, "/", 2)
			if len(parts) > 1 {
				folderName := parts[0]
				if !seenFolders[folderName] {
					folders = append(folders, folderName)
					seenFolders[folderName] = true
				}
			}
		}
	}

	return folders, nil
}

func (g *GCSClient) UploadFile(bucket, objectPath string, data []byte) error {
	if g.client == nil {
		return fmt.Errorf("GCS client not initialized properly")
	}

	writer := g.client.Bucket(bucket).Object(objectPath).NewWriter(g.ctx)
	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write file to GCS: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

func (g *GCSClient) CheckFileExists(bucket, objectPath string) (bool, error) {
	if g.client == nil {
		return false, fmt.Errorf("GCS client not initialized properly")
	}

	_, err := g.client.Bucket(bucket).Object(objectPath).Attrs(g.ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	return true, nil
}

func (g *GCSClient) CheckFolderExists(bucket, folderPrefix string) (bool, error) {
	if g.client == nil {
		return false, fmt.Errorf("GCS client not initialized properly")
	}

	// Ensure folderPrefix ends with "/"
	if !strings.HasSuffix(folderPrefix, "/") {
		folderPrefix += "/"
	}

	it := g.client.Bucket(bucket).Objects(g.ctx, &storage.Query{
		Prefix: folderPrefix,
	})

	// Check if at least one object exists with this prefix
	_, err := it.Next()
	if err == iterator.Done {
		return false, nil // No objects found with this prefix
	}
	if err != nil {
		return false, fmt.Errorf("failed to check folder existence: %w", err)
	}

	return true, nil
}

func (g *GCSClient) UploadFolderFromLocal(srcFolderPath, bucket, destPath string) error {
	if g.client == nil {
		return fmt.Errorf("GCS client not initialized properly")
	}

	// Walk through the source folder and upload all files
	return filepath.Walk(srcFolderPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking path %s: %w", filePath, err)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read file content
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		// Calculate relative path from source folder
		relPath, err := filepath.Rel(srcFolderPath, filePath)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Convert to forward slashes for GCS
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		// Upload to GCS
		gcsPath := path.Join(destPath, relPath)
		if err := g.UploadFile(bucket, gcsPath, data); err != nil {
			return fmt.Errorf("failed to upload file %s to GCS: %w", relPath, err)
		}

		log.Info().Msgf("Uploaded file: %s to gs://%s/%s", relPath, bucket, gcsPath)
		return nil
	})
}

func (g *GCSClient) GetFolderInfo(bucket, folderPrefix string) (*GCSFolderInfo, error) {
	if g.client == nil {
		return nil, fmt.Errorf("GCS client not initialized properly")
	}

	// Ensure folderPrefix ends with "/"
	if !strings.HasSuffix(folderPrefix, "/") {
		folderPrefix += "/"
	}

	it := g.client.Bucket(bucket).Objects(g.ctx, &storage.Query{
		Prefix: folderPrefix,
	})

	var folderInfo GCSFolderInfo
	folderInfo.Name = strings.TrimSuffix(path.Base(folderPrefix), "/")
	folderInfo.Path = fmt.Sprintf("gs://%s/%s", bucket, strings.TrimSuffix(folderPrefix, "/"))
	folderInfo.Created = time.Now()  // Will be updated with actual earliest file
	folderInfo.Updated = time.Time{} // Will be updated with actual latest file

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Update folder stats
		folderInfo.FileCount++
		folderInfo.Size += attrs.Size

		// Track earliest creation time
		if attrs.Created.Before(folderInfo.Created) {
			folderInfo.Created = attrs.Created
		}

		// Track latest update time
		if attrs.Updated.After(folderInfo.Updated) {
			folderInfo.Updated = attrs.Updated
		}
	}

	if folderInfo.FileCount == 0 {
		return nil, fmt.Errorf("folder not found or empty")
	}

	return &folderInfo, nil
}

func (g *GCSClient) ListFoldersWithTimestamp(bucket, prefix string) ([]GCSFolderInfo, error) {
	if g.client == nil {
		return nil, fmt.Errorf("GCS client not initialized properly")
	}

	var folders []GCSFolderInfo
	seenFolders := make(map[string]*GCSFolderInfo)

	// Ensure prefix ends with a trailing slash
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	log.Info().Msgf("Listing folders with timestamps in GCS bucket %s with prefix %s", bucket, prefix)

	it := g.client.Bucket(bucket).Objects(g.ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Extract folder name after the prefix
		if attrs.Name != "" {
			trimmed := strings.TrimPrefix(attrs.Name, prefix)
			parts := strings.SplitN(trimmed, "/", 2)
			if len(parts) > 1 {
				folderName := parts[0]

				// Initialize or update folder info
				if folderInfo, exists := seenFolders[folderName]; !exists {
					seenFolders[folderName] = &GCSFolderInfo{
						Name:      folderName,
						Path:      fmt.Sprintf("gs://%s/%s%s", bucket, prefix, folderName),
						Created:   attrs.Created,
						Updated:   attrs.Updated,
						Size:      attrs.Size,
						FileCount: 1,
					}
				} else {
					// Update existing folder info
					folderInfo.FileCount++
					folderInfo.Size += attrs.Size

					// Track earliest creation time
					if attrs.Created.Before(folderInfo.Created) {
						folderInfo.Created = attrs.Created
					}

					// Track latest update time
					if attrs.Updated.After(folderInfo.Updated) {
						folderInfo.Updated = attrs.Updated
					}
				}
			}
		}
	}

	// Convert map to slice
	for _, folderInfo := range seenFolders {
		folders = append(folders, *folderInfo)
	}

	return folders, nil
}

// FindFileWithSuffix finds the first file with the specified suffix in the given folder path
// Returns (exists, filename, error)
func (g *GCSClient) FindFileWithSuffix(bucket, folderPath, suffix string) (bool, string, error) {
	if g.client == nil {
		return false, "", fmt.Errorf("GCS client not initialized properly")
	}

	// Ensure folderPath ends with "/"
	if !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	log.Info().Msgf("Searching for files with suffix '%s' in GCS bucket %s with prefix %s", suffix, bucket, folderPath)

	it := g.client.Bucket(bucket).Objects(g.ctx, &storage.Query{
		Prefix: folderPath,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return false, "", fmt.Errorf("failed to list objects: %w", err)
		}

		// Get the filename from the full object path
		fileName := path.Base(attrs.Name)

		// Check if the file ends with the specified suffix
		if strings.HasSuffix(fileName, suffix) {
			log.Info().Msgf("Found file with suffix '%s': %s", suffix, fileName)
			return true, fileName, nil
		}
	}

	log.Info().Msgf("No file found with suffix '%s' in %s/%s", suffix, bucket, folderPath)
	return false, "", nil
}
