package externalcall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	TransferFolderWithSplitSources(modelBucket, modelPath, configBucket, configPath, srcModelName, destBuckt, destPath, destModelName string) error
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

func CreateGCSClient() GCSClientInterface {
	ctx := context.Background()

	// Check for Application Default Credentials path
	credsPath := os.Getenv("CLOUDSDK_CONFIG")
	if credsPath == "" {
		home := os.Getenv("HOME")
		if home == "" {
			home = "/root"
		}
		credsPath = home + "/.config/gcloud"
	}
	adcPath := credsPath + "/application_default_credentials.json"

	// Check if ADC file exists
	_, err := os.Stat(adcPath)
	adcExists := err == nil

	log.Info().
		Str("CLOUDSDK_CONFIG", os.Getenv("CLOUDSDK_CONFIG")).
		Str("HOME", os.Getenv("HOME")).
		Str("adcPath", adcPath).
		Bool("adcFileExists", adcExists).
		Msg("Creating GCS client with Application Default Credentials")

	if !adcExists {
		log.Warn().
			Str("adcPath", adcPath).
			Msg("ADC credentials file not found - GCS operations may fail. Run 'gcloud auth application-default login' on host and ensure ~/.config/gcloud is mounted")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Str("adcPath", adcPath).
			Bool("adcFileExists", adcExists).
			Msg("Failed to create GCS client - check ADC credentials")
		return &GCSClient{
			client: nil,
			ctx:    ctx,
		}
	}

	// Log successful creation with credential info
	log.Info().
		Str("adcPath", adcPath).
		Bool("adcFileExists", adcExists).
		Msg("GCS client created successfully with Application Default Credentials")

	return &GCSClient{
		client: client,
		ctx:    ctx,
	}
}

// ObjectVisitor is called for each object. Return an error to stop iteration.
// Return a special sentinel error like ErrStopIteration to stop without error.
type ObjectVisitor func(attrs *storage.ObjectAttrs) error

var ErrStopIteration = errors.New("stop iteration")

// ObjectFilter returns true if the object should be included.
type ObjectFilter func(attrs *storage.ObjectAttrs) bool

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

	isConfigFile := func(attrs *storage.ObjectAttrs) bool {
		return strings.HasSuffix(attrs.Name, "config.pbtxt")
	}

	configFiles, regularFiles, err := g.partitionObjects(srcBucket, prefix, isConfigFile)
	if err != nil {
		return fmt.Errorf("failed to list source bucket: %w", err)
	}

	if len(regularFiles) == 0 && len(configFiles) == 0 {
		return fmt.Errorf("no files found at source location: gs://%s/%s", srcBucket, prefix)
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

func (g *GCSClient) TransferFolderWithSplitSources(modelBucket, modelPath, configBucket, configPath, srcModelName, destBucket, destPath, destModelName string) error {
	modelPrefix := path.Join(modelPath, srcModelName)
	if !strings.HasSuffix(modelPrefix, "/") {
		modelPrefix += "/"
	}

	regularFiles, err := g.listObjects(modelBucket, modelPrefix, func(attrs *storage.ObjectAttrs) bool {
		return !strings.HasSuffix(attrs.Name, "config.pbtxt")
	})
	if err != nil {
		return fmt.Errorf("failed to read regular files from model source: %w", err)
	}

	log.Info().Msgf("TransferFolderWithSplitSources: Found %d regular files in model source gs://%s/%s",
		len(regularFiles), modelBucket, modelPrefix)

	configPrefix := path.Join(configPath, srcModelName)
	if !strings.HasSuffix(configPrefix, "/") {
		configPrefix += "/"
	}

	configFiles, err := g.listObjects(configBucket, configPrefix, func(attrs *storage.ObjectAttrs) bool {
		return strings.HasSuffix(attrs.Name, "config.pbtxt")
	})
	if err != nil {
		return fmt.Errorf("failed to read config files from config source: %w", err)
	}

	log.Info().Msgf("TransferFolderWithSplitSources: Found %d config files in config source gs://%s/%s",
		len(configFiles), configBucket, configPrefix)

	if len(regularFiles) == 0 && len(configFiles) == 0 {
		return fmt.Errorf("transferFolderWithSplitSources: No objects found to transfer")
	}

	regularFilesTransferred := false
	if len(regularFiles) > 0 {
		if err := g.transferRegularFilesFromSource(regularFiles, modelBucket, destBucket, destPath, destModelName, modelPrefix); err != nil {
			return fmt.Errorf("failed to transfer regular files from model source: %w", err)
		}
		regularFilesTransferred = true
	}

	if len(configFiles) > 0 {
		if err := g.transferConfigFilesFromSource(configFiles, configBucket, destBucket, destPath, destModelName, configPrefix); err != nil {
			if regularFilesTransferred {
				log.Warn().Err(err).Msgf("Config file transfer failed, reverting transfer by deleting destination folder gs://%s/%s/%s",
					destBucket, destPath, destModelName)
				if revertErr := g.DeleteFolder(destBucket, destPath, destModelName); revertErr != nil {
					log.Error().Err(revertErr).Msgf("Failed to revert transfer by deleting destination folder gs://%s/%s/%s",
						destBucket, destPath, destModelName)
					return fmt.Errorf("failed to transfer config files from config source: %w; revert also failed: %w", err, revertErr)
				}
				log.Info().Msgf("Successfully reverted transfer by deleting destination folder")
			}
			return fmt.Errorf("failed to transfer config files from config source: %w", err)
		}
	}

	log.Info().Msgf("TransferFolderWithSplitSources: Successfully completed split-source transfer for model %s -> %s",
		srcModelName, destModelName)
	return nil
}

func (g *GCSClient) transferRegularFilesFromSource(files []storage.ObjectAttrs, srcBucket, destBucket, destPath, destModelName, prefix string) error {
	log.Info().Msgf("Transferring %d regular files from model source", len(files))

	semaphore := make(chan struct{}, maxConcurrentFiles)
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, objAttrs := range files {
		wg.Add(1)
		go func(obj storage.ObjectAttrs) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := g.transferSingleRegularFile(obj, srcBucket, destBucket, destPath, destModelName, prefix); err != nil {
				errChan <- fmt.Errorf("failed to transfer %s: %w", obj.Name, err)
			}
		}(objAttrs)
	}

	wg.Wait()

	if errCount := len(errChan); errCount > 0 {
		errs := make([]error, 0, errCount)
		for i := 0; i < errCount; i++ {
			errs = append(errs, <-errChan)
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("regular file transfer completed with %d errors:\n", len(errs)))
		for i, e := range errs {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(e.Error())
		}
		return fmt.Errorf("%s", b.String())
	}

	return nil
}

func (g *GCSClient) transferConfigFilesFromSource(files []storage.ObjectAttrs, srcBucket, destBucket, destPath, destModelName, prefix string) error {
	log.Info().Msgf("Transferring %d config files from config source", len(files))

	for _, objAttrs := range files {
		if err := g.transferSingleConfigFile(objAttrs, srcBucket, destBucket, destPath, destModelName, prefix); err != nil {
			return fmt.Errorf("failed to transfer config file %s: %w", objAttrs.Name, err)
		}
	}

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

// replaceModelNameInConfig modifies only the top-level `name:` field in config.pbtxt content
// It replaces only the first occurrence to avoid modifying nested names in inputs/outputs/instance_groups
func replaceModelNameInConfig(data []byte, destModelName string) []byte {
	content := string(data)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match top-level "name:" field - should be at the start of line (or minimal indentation)
		// Skip nested names which are typically indented with 2+ spaces
		if strings.HasPrefix(trimmed, "name:") {
			// Check indentation: top-level fields have minimal/no indentation
			leadingWhitespace := len(line) - len(strings.TrimLeft(line, " \t"))
			// Skip if heavily indented (nested field)
			if leadingWhitespace >= 2 {
				continue
			}

			// Match the first occurrence of name: "value" pattern
			namePattern := regexp.MustCompile(`name\s*:\s*"([^"]+)"`)
			matches := namePattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				oldModelName := matches[1]
				// Replace only the FIRST occurrence to avoid replacing nested names
				loc := namePattern.FindStringIndex(line)
				if loc != nil {
					// Replace only the matched portion (first occurrence)
					before := line[:loc[0]]
					matched := line[loc[0]:loc[1]]
					after := line[loc[1]:]
					// Replace the value inside quotes while preserving the "name:" format
					valuePattern := regexp.MustCompile(`"([^"]+)"`)
					valueReplaced := valuePattern.ReplaceAllString(matched, fmt.Sprintf(`"%s"`, destModelName))
					lines[i] = before + valueReplaced + after
				} else {
					// Fallback: replace all (shouldn't happen with valid input)
					lines[i] = namePattern.ReplaceAllString(line, fmt.Sprintf(`name: "%s"`, destModelName))
				}
				log.Info().Msgf("Replacing top-level model name in config.pbtxt: '%s' -> '%s'", oldModelName, destModelName)
				break
			}
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

	err := g.forEachObject(bucket, prefix, func(attrs *storage.ObjectAttrs) error {
		if attrs.Name == "" {
			return nil
		}
		trimmed := strings.TrimPrefix(attrs.Name, prefix)
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) > 1 {
			folderName := parts[0]
			if !seenFolders[folderName] {
				folders = append(folders, folderName)
				seenFolders[folderName] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
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

	var exists bool
	err := g.forEachObject(bucket, folderPrefix, func(attrs *storage.ObjectAttrs) error {
		exists = true
		return ErrStopIteration // Found one, stop iteration
	})
	if err != nil {
		return false, fmt.Errorf("failed to check folder existence: %w", err)
	}
	return exists, nil
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

	var folderInfo GCSFolderInfo
	folderInfo.Name = strings.TrimSuffix(path.Base(folderPrefix), "/")
	folderInfo.Path = fmt.Sprintf("gs://%s/%s", bucket, strings.TrimSuffix(folderPrefix, "/"))
	folderInfo.Created = time.Now()
	folderInfo.Updated = time.Time{}

	err := g.forEachObject(bucket, folderPrefix, func(attrs *storage.ObjectAttrs) error {
		folderInfo.FileCount++
		folderInfo.Size += attrs.Size

		if attrs.Created.Before(folderInfo.Created) {
			folderInfo.Created = attrs.Created
		}
		if attrs.Updated.After(folderInfo.Updated) {
			folderInfo.Updated = attrs.Updated
		}
		return nil
	})
	if err != nil {
		return nil, err
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
	err := g.forEachObject(bucket, prefix, func(attrs *storage.ObjectAttrs) error {
		if attrs.Name == "" {
			return nil
		}
		trimmed := strings.TrimPrefix(attrs.Name, prefix)
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) > 1 {
			folderName := parts[0]

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
				folderInfo.FileCount++
				folderInfo.Size += attrs.Size
				if attrs.Created.Before(folderInfo.Created) {
					folderInfo.Created = attrs.Created
				}
				if attrs.Updated.After(folderInfo.Updated) {
					folderInfo.Updated = attrs.Updated
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
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

	var foundFile string
	err := g.forEachObject(bucket, folderPath, func(attrs *storage.ObjectAttrs) error {
		fileName := path.Base(attrs.Name)
		if strings.HasSuffix(fileName, suffix) {
			log.Info().Msgf("Found file with suffix '%s': %s", suffix, fileName)
			foundFile = fileName
			return ErrStopIteration
		}
		return nil
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to list objects: %w", err)
	}
	if foundFile == "" {
		return false, "", fmt.Errorf("no file found with suffix '%s' in %s/%s", suffix, bucket, folderPath)
	}
	return true, foundFile, nil
}

// forEachObject iterates over all objects with the given prefix and calls the visitor for each.
func (g *GCSClient) forEachObject(bucket, prefix string, visitor ObjectVisitor) error {
	if g.client == nil {
		return fmt.Errorf("GCS client not initialized properly")
	}

	it := g.client.Bucket(bucket).Objects(g.ctx, &storage.Query{Prefix: prefix})
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		if err := visitor(objAttrs); err != nil {
			if errors.Is(err, ErrStopIteration) {
				return nil
			}
			return err
		}
	}
	return nil
}

// listObjects returns all objects matching the prefix, optionally filtered.
// Pass nil for filter to include all objects (except directory markers).
func (g *GCSClient) listObjects(bucket, prefix string, filter ObjectFilter) ([]storage.ObjectAttrs, error) {
	if g.client == nil {
		return nil, fmt.Errorf("GCS client not initialized properly")
	}

	var objects []storage.ObjectAttrs

	err := g.forEachObject(bucket, prefix, func(attrs *storage.ObjectAttrs) error {
		// Skip directory markers by default
		if strings.HasSuffix(attrs.Name, "/") {
			return nil
		}

		if filter == nil || filter(attrs) {
			objects = append(objects, *attrs)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return objects, nil
}

// partitionObjects separates objects into two groups based on a predicate.
// Objects matching the predicate go into the first slice, others into the second.
func (g *GCSClient) partitionObjects(bucket, prefix string, predicate ObjectFilter) (matching, notMatching []storage.ObjectAttrs, err error) {
	if g.client == nil {
		return nil, nil, fmt.Errorf("GCS client not initialized properly")
	}

	err = g.forEachObject(bucket, prefix, func(attrs *storage.ObjectAttrs) error {
		// Skip directory markers
		if strings.HasSuffix(attrs.Name, "/") {
			return nil
		}

		if predicate(attrs) {
			matching = append(matching, *attrs)
		} else {
			notMatching = append(notMatching, *attrs)
		}
		return nil
	})

	return matching, notMatching, err
}
