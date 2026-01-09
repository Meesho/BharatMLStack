package asyncloguploader

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/storage"
)

// ChunkManager handles GCS compose operations with 32-chunk limit
type ChunkManager struct {
	maxChunksPerCompose int // Default: 32 (GCS limit)
}

// NewChunkManager creates a new chunk manager
func NewChunkManager(maxChunksPerCompose int) *ChunkManager {
	if maxChunksPerCompose <= 0 {
		maxChunksPerCompose = 32 // GCS limit
	}
	return &ChunkManager{
		maxChunksPerCompose: maxChunksPerCompose,
	}
}

// Compose composes chunks into final object, handling 32-chunk limit
func (cm *ChunkManager) Compose(ctx context.Context, client *storage.Client,
	bucket, object string, chunkObjects []string) error {

	if len(chunkObjects) <= cm.maxChunksPerCompose {
		// Single compose operation
		return cm.singleCompose(ctx, client, bucket, object, chunkObjects)
	}

	// Multi-level compose needed
	return cm.multiLevelCompose(ctx, client, bucket, object, chunkObjects)
}

// singleCompose performs a single compose operation (chunks <= 32)
func (cm *ChunkManager) singleCompose(ctx context.Context, client *storage.Client,
	bucket, object string, chunkObjects []string) error {

	if len(chunkObjects) == 0 {
		return fmt.Errorf("no chunks to compose")
	}

	if len(chunkObjects) > cm.maxChunksPerCompose {
		return fmt.Errorf("too many chunks (%d), max is %d", len(chunkObjects), cm.maxChunksPerCompose)
	}

	bkt := client.Bucket(bucket)
	dst := bkt.Object(object)

	// Build list of source objects in order
	sources := make([]*storage.ObjectHandle, len(chunkObjects))
	for i, chunkObj := range chunkObjects {
		sources[i] = bkt.Object(chunkObj)
	}

	// Compose: GCS atomically combines all chunks in order
	composer := dst.ComposerFrom(sources...)
	composer.ContentType = "application/octet-stream"

	_, err := composer.Run(ctx)
	if err != nil {
		return fmt.Errorf("compose failed: %w", err)
	}

	return nil
}

// multiLevelCompose performs multi-level compose for files with >32 chunks
func (cm *ChunkManager) multiLevelCompose(ctx context.Context, client *storage.Client,
	bucket, object string, chunkObjects []string) error {

	// Compose groups of maxChunksPerCompose into intermediate objects
	var intermediateObjects []string
	for i := 0; i < len(chunkObjects); i += cm.maxChunksPerCompose {
		end := i + cm.maxChunksPerCompose
		if end > len(chunkObjects) {
			end = len(chunkObjects)
		}

		group := chunkObjects[i:end]
		// Generate intermediate object name with full path to avoid conflicts
		// Use the same prefix/directory structure as the final object
		intermediateObj := fmt.Sprintf("%s.intermediate.%d", object, i/cm.maxChunksPerCompose)

		if err := cm.singleCompose(ctx, client, bucket, intermediateObj, group); err != nil {
			// Cleanup any intermediate objects created so far
			cm.cleanupObjects(ctx, client, bucket, intermediateObjects)
			return fmt.Errorf("failed to compose intermediate object %s: %w", intermediateObj, err)
		}

		intermediateObjects = append(intermediateObjects, intermediateObj)
	}

	// Recursively compose intermediate objects if needed
	if len(intermediateObjects) <= cm.maxChunksPerCompose {
		// Final compose
		if err := cm.singleCompose(ctx, client, bucket, object, intermediateObjects); err != nil {
			cm.cleanupObjects(ctx, client, bucket, intermediateObjects)
			return err
		}
		// Cleanup intermediate objects after successful final compose
		cm.cleanupObjects(ctx, client, bucket, intermediateObjects)
		return nil
	}

	// Need another level of compose
	if err := cm.multiLevelCompose(ctx, client, bucket, object, intermediateObjects); err != nil {
		cm.cleanupObjects(ctx, client, bucket, intermediateObjects)
		return err
	}

	// Cleanup intermediate objects after successful compose
	cm.cleanupObjects(ctx, client, bucket, intermediateObjects)
	return nil
}

// cleanupObjects deletes objects (for cleanup on error or after use)
func (cm *ChunkManager) cleanupObjects(ctx context.Context, client *storage.Client,
	bucket string, objects []string) {

	bkt := client.Bucket(bucket)
	for _, obj := range objects {
		if err := bkt.Object(obj).Delete(ctx); err != nil {
			log.Printf("[WARNING] Failed to cleanup object %s: %v", obj, err)
		}
	}
}
