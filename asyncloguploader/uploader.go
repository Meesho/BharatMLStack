package asyncloguploader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	logger "github.com/rs/zerolog/log"
	"google.golang.org/api/option"

	"github.com/Meesho/go-core/metric"
)

// Note: GCSUploadConfig is now defined in config.go
// This file uses GCSUploadConfig from the config package

// filenamePartitionRegex matches {eventname}_{YYYY-MM-DD_HH-MM-SS}.log
var filenamePartitionRegex = regexp.MustCompile(`^(.+)_(\d{4}-\d{2}-\d{2})_(\d{2})-\d{2}-\d{2}\.log$`)

// Uploader handles uploading completed log files to GCS by scanning the log directory
type Uploader struct {
	config       GCSUploadConfig
	client       *storage.Client
	scanDir      string
	scanInterval time.Duration
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	metricTags   []string
	chunkMgr     *ChunkManager
	stopOnce     sync.Once // Ensures Stop() is idempotent
}

// NewUploader creates a new GCS uploader service that scans scanDir for .log files
// metricTags are application-provided tags for metric emissions (e.g., from Config.MetricTags)
func NewUploader(config GCSUploadConfig, scanDir string, metricTags []string) (*Uploader, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create GCS client with gRPC pool
	client, err := storage.NewClient(ctx,
		option.WithGRPCConnectionPool(config.GRPCPoolSize),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	interval := config.ScanInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}

	tags := getBaseTags(metricTags)
	uploader := &Uploader{
		config:       config,
		client:       client,
		scanDir:      scanDir,
		scanInterval: interval,
		ctx:          ctx,
		cancel:       cancel,
		metricTags:   tags,
		chunkMgr:     NewChunkManager(config.MaxChunksPerCompose),
	}

	return uploader, nil
}

// Start starts the uploader service (scans scanDir for .log files and uploads)
func (u *Uploader) Start() {
	u.wg.Add(1)
	go u.uploadWorker()
}

// Stop stops the uploader service gracefully
// Safe to call multiple times (idempotent)
func (u *Uploader) Stop() {
	u.stopOnce.Do(func() {
		// Cancel context to signal worker to stop
		u.cancel()

		// Wait for upload worker to finish (does final scan before exiting)
		u.wg.Wait()

		// Close client
		u.client.Close()
	})
}

// uploadWorker scans scanDir periodically for .log files and uploads them
func (u *Uploader) uploadWorker() {
	defer u.wg.Done()

	ticker := time.NewTicker(u.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-u.ctx.Done():
			// Final scan before exiting
			u.scanAndUpload()
			logger.Debug().Msg("Upload worker exiting (context cancelled)")
			return
		case <-ticker.C:
			u.scanAndUpload()
		}
	}
}

// scanAndUpload lists .log files in scanDir and uploads each
func (u *Uploader) scanAndUpload() {
	entries, err := os.ReadDir(u.scanDir)
	if err != nil {
		logger.Debug().Err(err).Msgf("Failed to read scan dir %s", u.scanDir)
		return
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		path := filepath.Join(u.scanDir, e.Name())

		logger.Debug().Msgf("Processing file for upload: %s", path)

		if err := u.uploadFileWithRetry(path); err != nil {
			logger.Error().Err(err).Msgf("Failed to upload %s after %d retries", path, u.config.MaxRetries)
			metric.Incr(MetricUploadFileFailed, u.metricTags)
		} else {
			logger.Debug().Msgf("Successfully uploaded: %s", path)
		}
	}
}

// uploadFileWithRetry uploads a file with retry logic
func (u *Uploader) uploadFileWithRetry(filePath string) error {
	// Get file size BEFORE upload (file will be deleted after successful upload)
	metric.Incr(MetricUploadFile, u.metricTags)
	fileInfo, statErr := os.Stat(filePath)
	var fileSize int64
	if statErr == nil {
		fileSize = fileInfo.Size()
	}

	var lastErr error
	for attempt := 0; attempt <= u.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-u.ctx.Done():
				return fmt.Errorf("uploader stopped")
			case <-time.After(u.config.RetryDelay):
			}
		}

		start := time.Now()
		err := u.uploadFile(filePath)
		metric.TimingWithStart(MetricUploadFileDuration, start, u.metricTags)

		if err == nil {
			// Success - emit bytes metric
			if statErr == nil && fileSize > 0 {
				metric.Count(MetricUploadBytes, fileSize, u.metricTags)
			}
			return nil
		}

		lastErr = err
		if attempt < u.config.MaxRetries {
			logger.Warn().Err(err).Msgf(
				"Upload attempt %d/%d failed for %s, retrying...",
				attempt+1,
				u.config.MaxRetries+1,
				filePath,
			)
		}
	}

	return fmt.Errorf("upload failed after %d attempts: %w", u.config.MaxRetries+1, lastErr)
}

// uploadFile uploads a single file to GCS using parallel chunk upload
func (u *Uploader) uploadFile(filePath string) error {
	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	// Read entire file into memory (for parallel chunk upload)
	// Note: For very large files, consider streaming instead
	buf := make([]byte, fileSize)
	if _, err := io.ReadFull(file, buf); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Generate object name
	objectName := u.generateObjectName(filePath)

	// Upload using parallel chunk upload with chunk manager
	if err := u.uploadParallel(u.ctx, u.client, u.config.Bucket, objectName, buf, u.config.ChunkSize); err != nil {
		return fmt.Errorf("parallel upload failed: %w", err)
	}

	// Clear buffer reference to help GC (buf will be garbage collected after function returns)
	buf = nil

	// Delete local file after successful upload
	if err := os.Remove(filePath); err != nil {
		logger.Warn().Err(err).Msgf("Failed to delete local file %s after upload", filePath)
		// Non-fatal - upload succeeded
	}

	return nil
}

// generateObjectName generates the GCS object name from file path.
// Partitions by eventname/date/hour when filename matches {event}_{YYYY-MM-DD_HH-MM-SS}.log.
// Fallback: flat {ObjectPrefix}{filename}
func (u *Uploader) generateObjectName(filePath string) string {
	fileName := filepath.Base(filePath)
	matches := filenamePartitionRegex.FindStringSubmatch(fileName)
	if len(matches) == 4 {
		eventName := matches[1]
		date := matches[2] // YYYY-MM-DD
		hour := matches[3] // HH
		prefix := u.config.ObjectPrefix
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix = prefix + "/"
		}
		return fmt.Sprintf("%s%s/%s/%s/%s", prefix, eventName, date, hour, fileName)
	}
	// Fallback: flat structure
	if u.config.ObjectPrefix != "" {
		return fmt.Sprintf("%s%s", u.config.ObjectPrefix, fileName)
	}
	return fileName
}

// uploadParallel uploads chunks in parallel and composes them into the final object
// This is based on the existing gcs_uploader module
func (u *Uploader) uploadParallel(ctx context.Context, client *storage.Client, bucket, object string, buf []byte, chunkSizeBytes int) error {
	// Calculate number of chunks
	numChunks := (len(buf) + chunkSizeBytes - 1) / chunkSizeBytes

	// Generate unique prefix for temporary chunk objects
	uploadID := time.Now().UnixNano()
	tempPrefix := fmt.Sprintf("%s.tmp.%d", object, uploadID)

	// Track chunk uploads
	type chunkResult struct {
		index  int
		object string
		size   int64
		err    error
	}

	results := make([]chunkResult, numChunks)
	var wg sync.WaitGroup

	// Upload chunks in parallel
	for i := 0; i < numChunks; i++ {
		offset := i * chunkSizeBytes
		end := offset + chunkSizeBytes
		if end > len(buf) {
			end = len(buf)
		}

		wg.Add(1)
		go func(chunkIndex int, chunkData []byte) {
			defer wg.Done()

			chunkObject := fmt.Sprintf("%s.chunk.%d", tempPrefix, chunkIndex)

			// Upload this chunk as a separate object
			w := client.Bucket(bucket).Object(chunkObject).NewWriter(ctx)
			w.ChunkSize = chunkSizeBytes
			w.ContentType = "application/octet-stream"

			if _, err := w.Write(chunkData); err != nil {
				results[chunkIndex] = chunkResult{
					index: chunkIndex,
					err:   fmt.Errorf("write error: %w", err),
				}
				return
			}

			if err := w.Close(); err != nil {
				results[chunkIndex] = chunkResult{
					index: chunkIndex,
					err:   fmt.Errorf("close error: %w", err),
				}
				return
			}

			// Get object attributes to verify size
			attrs, err := client.Bucket(bucket).Object(chunkObject).Attrs(ctx)
			if err != nil {
				results[chunkIndex] = chunkResult{
					index: chunkIndex,
					err:   fmt.Errorf("attrs error: %w", err),
				}
				return
			}

			results[chunkIndex] = chunkResult{
				index:  chunkIndex,
				object: chunkObject,
				size:   attrs.Size,
			}
		}(i, buf[offset:end])
	}

	// Wait for all uploads to complete
	wg.Wait()

	// Check for errors
	for _, result := range results {
		if result.err != nil {
			// Cleanup: delete any successfully uploaded chunks
			u.cleanupTempChunks(ctx, client, bucket, tempPrefix, numChunks)
			return fmt.Errorf("chunk %d failed: %w", result.index, result.err)
		}
	}

	// Build list of chunk object names
	chunkObjects := make([]string, numChunks)
	for i := 0; i < numChunks; i++ {
		chunkObjects[i] = fmt.Sprintf("%s.chunk.%d", tempPrefix, i)
	}

	// Use chunk manager to compose (handles 32-chunk limit)
	if err := u.chunkMgr.Compose(ctx, client, bucket, object, chunkObjects); err != nil {
		// Cleanup on failure
		u.cleanupTempChunks(ctx, client, bucket, tempPrefix, numChunks)
		logger.Error().Err(err).Msgf("Compose failed for %s (%d chunks). Chunks may remain in GCS.", object, numChunks)
		return fmt.Errorf("compose error: %w", err)
	}

	// Log successful compose for debugging
	if numChunks > 1 {
		logger.Debug().Msgf("Successfully composed %d chunks into %s", numChunks, object)
	}

	// Verify final object size matches expected size
	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		u.cleanupTempChunks(ctx, client, bucket, tempPrefix, numChunks)
		return fmt.Errorf("failed to get object attributes: %w", err)
	}

	if attrs.Size != int64(len(buf)) {
		// Cleanup and return error
		u.cleanupTempChunks(ctx, client, bucket, tempPrefix, numChunks)
		_ = client.Bucket(bucket).Object(object).Delete(ctx) // Try to delete malformed object
		return fmt.Errorf("size mismatch: expected %d bytes, got %d bytes", len(buf), attrs.Size)
	}

	// Cleanup temporary chunk objects
	if err := u.cleanupTempChunks(ctx, client, bucket, tempPrefix, numChunks); err != nil {
		logger.Warn().Err(err).Msg("Failed to cleanup some temp chunks")
		// Non-fatal - main upload succeeded
	}

	return nil
}

// cleanupTempChunks deletes temporary chunk objects
func (u *Uploader) cleanupTempChunks(ctx context.Context, client *storage.Client, bucket, prefix string, numChunks int) error {
	var errs []error
	bkt := client.Bucket(bucket)

	for i := 0; i < numChunks; i++ {
		chunkObject := fmt.Sprintf("%s.chunk.%d", prefix, i)
		if err := bkt.Object(chunkObject).Delete(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete %s: %w", chunkObject, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}
