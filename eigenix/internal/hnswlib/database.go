package hnswlib

/*
#cgo CXXFLAGS: -std=c++11 -I${SRCDIR}/../../pkg/hnswlib
#cgo LDFLAGS: -lstdc++
#include "../../pkg/hnswlib/hnsw_wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	pb "github.com/Meesho/skye-eigenix/internal/client"
)

const (
	// Directory to store persisted indices
	INDICES_DIR = "./indices"
	// Metadata file extension
	METADATA_EXT = ".meta"
	// Index file extension
	INDEX_EXT = ".idx"
)

// HNSWIndex represents an in-memory HNSW index
type HNSWIndex struct {
	Name                string      `json:"name"`
	Dimension           int32       `json:"dimension"`
	Space               string      `json:"space"` // "l2", "ip", "cosine"
	MaxElements         int32       `json:"max_elements"`
	M                   int32       `json:"m"`
	EfConstruct         int32       `json:"ef_construction"`
	AllowReplaceDeleted bool        `json:"allow_replace_deleted"`
	CIndex              C.HNSWIndex `json:"-"` // C++ HNSW index pointer
}

// searchBuffer holds pre-allocated C memory for search operations
type searchBuffer struct {
	k         int
	labels    *C.ulonglong
	distances *C.float
}

type job struct {
	idx   int
	query []float32
}

// Error handling
type result struct {
	idx     int
	results []SearchResult
	err     error
}

// Database implements the Manager interface
type Database struct {
	indices     map[string]*HNSWIndex
	mutex       sync.RWMutex
	bufferPools map[int]*sync.Pool // Pools of search buffers by k value
	poolsMutex  sync.RWMutex
}

var (
	database Manager
	once     sync.Once
)

// Init initializes and returns the singleton database instance
func Init() Manager {
	if database == nil {
		once.Do(func() {
			database = &Database{
				indices:     make(map[string]*HNSWIndex),
				bufferPools: make(map[int]*sync.Pool),
			}
		})
	}
	return database
}

// getSearchBuffer retrieves or creates a search buffer from the pool
func (db *Database) getSearchBuffer(k int) *searchBuffer {
	db.poolsMutex.RLock()
	pool, exists := db.bufferPools[k]
	db.poolsMutex.RUnlock()

	if !exists {
		// Create a new pool for this k value
		db.poolsMutex.Lock()
		// Double-check after acquiring write lock
		pool, exists = db.bufferPools[k]
		if !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					return &searchBuffer{
						k:         k,
						labels:    (*C.ulonglong)(C.malloc(C.size_t(k) * C.size_t(unsafe.Sizeof(C.ulonglong(0))))),
						distances: (*C.float)(C.malloc(C.size_t(k) * C.size_t(unsafe.Sizeof(C.float(0))))),
					}
				},
			}
			db.bufferPools[k] = pool
		}
		db.poolsMutex.Unlock()
	}

	return pool.Get().(*searchBuffer)
}

// putSearchBuffer returns a search buffer to the pool
func (db *Database) putSearchBuffer(buf *searchBuffer) {
	db.poolsMutex.RLock()
	pool, exists := db.bufferPools[buf.k]
	db.poolsMutex.RUnlock()

	if exists {
		pool.Put(buf)
	}
}

// CreateIndex creates a new HNSW index
func (db *Database) CreateIndex(name, space string, dimension int32, maxElements int32, m int32, efConstruct int32, allowReplaceDeleted bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if index already exists
	if _, exists := db.indices[name]; exists {
		return fmt.Errorf("index already exists")
	}

	// Create new HNSW index using C++ implementation
	cSpaceName := C.CString(space)
	defer C.free(unsafe.Pointer(cSpaceName))

	var cAllowReplace C.int
	if allowReplaceDeleted {
		cAllowReplace = 1
	} else {
		cAllowReplace = 0
	}

	cIndex := C.hnsw_create_index(
		cSpaceName,
		C.int(dimension),
		C.int(maxElements),
		C.int(m),
		C.int(efConstruct),
		C.int(100), // random seed
		cAllowReplace,
	)

	if cIndex == nil {
		return fmt.Errorf("failed to create HNSW index")
	}

	// Create new index
	index := &HNSWIndex{
		Name:                name,
		Dimension:           dimension,
		Space:               space,
		MaxElements:         maxElements,
		M:                   m,
		EfConstruct:         efConstruct,
		AllowReplaceDeleted: allowReplaceDeleted,
		CIndex:              cIndex,
	}

	db.indices[name] = index
	return nil
}

// GetIndex retrieves an index by name
func (db *Database) GetIndex(name string) (*HNSWIndex, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	index, exists := db.indices[name]
	return index, exists
}

// ListIndices returns all indices
func (db *Database) ListIndices() []*HNSWIndex {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	indices := make([]*HNSWIndex, 0, len(db.indices))
	for _, index := range db.indices {
		indices = append(indices, index)
	}
	return indices
}

// DeleteIndex removes an index from memory
func (db *Database) DeleteIndex(name string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	index, exists := db.indices[name]
	if !exists {
		return fmt.Errorf("index not found")
	}

	// Clean up C++ resources
	if index.CIndex != nil {
		C.hnsw_delete_index(index.CIndex)
	}

	delete(db.indices, name)
	return nil
}

// AddPoint adds a single point to an index (pure database operation)
func (db *Database) AddPoint(indexName string, id uint64, vector []float32) error {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("index not found")
	}

	// Convert Go slice to C array
	cVector := (*C.float)(unsafe.Pointer(&vector[0]))

	// Add point to HNSW index
	result := C.hnsw_add_point(index.CIndex, cVector, C.ulonglong(id))
	if result != 0 {
		return fmt.Errorf("failed to add point")
	}

	return nil
}

// AddPoints adds multiple points to an index (pure database operation)
// Uses parallel processing for batches > 100 points
func (db *Database) AddPoints(indexName string, points []*pb.Point) (int, error) {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return 0, fmt.Errorf("index not found")
	}

	numWorkers := runtime.NumCPU() / 4
	var wg sync.WaitGroup
	var successCount atomic.Int32

	// Create work queue
	jobs := make(chan *pb.Point, len(points))

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for point := range jobs {
				if point == nil || len(point.Vector) == 0 {
					continue
				}
				cVector := (*C.float)(unsafe.Pointer(&point.Vector[0]))
				result := C.hnsw_add_point(index.CIndex, cVector, C.ulonglong(point.Id))
				if result == 0 {
					successCount.Add(1)
				}
			}
		}()
	}

	for _, point := range points {
		jobs <- point
	}
	close(jobs)

	wg.Wait()

	return int(successCount.Load()), nil
}

// Search performs k-NN search on an index (pure database operation)
func (db *Database) Search(indexName string, query []float32, k int) ([]SearchResult, error) {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index not found")
	}

	if index.CIndex == nil {
		return []SearchResult{}, fmt.Errorf("index not initialized")
	}

	// Convert Go slice to C array
	cQuery := (*C.float)(unsafe.Pointer(&query[0]))

	// Get reusable buffer from pool
	buf := db.getSearchBuffer(k)
	defer db.putSearchBuffer(buf)

	// Perform HNSW search
	resultCount := int(C.hnsw_search_knn(index.CIndex, cQuery, C.int(k), buf.labels, buf.distances))

	// Convert C arrays back to Go slices
	results := make([]SearchResult, resultCount)
	labelsSlice := (*[1 << 30]C.ulonglong)(unsafe.Pointer(buf.labels))[:resultCount:resultCount]
	distancesSlice := (*[1 << 30]C.float)(unsafe.Pointer(buf.distances))[:resultCount:resultCount]

	for i := 0; i < resultCount; i++ {
		distance := distancesSlice[i]
		if index.Space == "cosine" {
			distance = 1 - distancesSlice[i]
		}
		results[i] = SearchResult{
			ID:       uint64(labelsSlice[i]),
			Distance: float32(distance),
		}
	}

	return results, nil
}

// BatchSearch performs k-NN search for multiple query vectors in parallel (pure database operation)
func (db *Database) BatchSearch(indexName string, queries []*pb.Vector, k int) ([][]SearchResult, error) {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index not found")
	}

	if index.CIndex == nil {
		return nil, fmt.Errorf("index not initialized")
	}

	numQueries := len(queries)
	results := make([][]SearchResult, numQueries)

	// For large batches, use parallel processing with worker pool
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	jobs := make(chan job, numQueries)

	resultsChan := make(chan result, numQueries)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				cQuery := (*C.float)(unsafe.Pointer(&j.query[0]))

				// Get reusable buffer from pool
				buf := db.getSearchBuffer(k)

				// Perform HNSW search
				resultCount := int(C.hnsw_search_knn(index.CIndex, cQuery, C.int(k), buf.labels, buf.distances))

				// Convert C arrays back to Go slices
				queryResults := make([]SearchResult, resultCount)
				labelsSlice := (*[1 << 30]C.ulonglong)(unsafe.Pointer(buf.labels))[:resultCount:resultCount]
				distancesSlice := (*[1 << 30]C.float)(unsafe.Pointer(buf.distances))[:resultCount:resultCount]

				for i := 0; i < resultCount; i++ {
					distance := distancesSlice[i]
					if index.Space == "cosine" {
						distance = 1 - distancesSlice[i]
					}
					queryResults[i] = SearchResult{
						ID:       uint64(labelsSlice[i]),
						Distance: float32(distance),
					}
				}

				// Return buffer to pool
				db.putSearchBuffer(buf)

				resultsChan <- result{idx: j.idx, results: queryResults, err: nil}
			}
		}()
	}

	// Send jobs to workers
	for i, v := range queries {
		jobs <- job{idx: i, query: v.Values}
	}
	close(jobs)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for r := range resultsChan {
		if r.err != nil {
			return nil, fmt.Errorf("query %d failed: %v", r.idx, r.err)
		}
		results[r.idx] = r.results
	}

	return results, nil
}

// GetIndexStats returns statistics about an index
func (db *Database) GetIndexStats(indexName string) (int, error) {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return 0, fmt.Errorf("index not found")
	}

	return int(C.hnsw_get_current_count(index.CIndex)), nil
}

// UpdatePoint updates a single point in an index (pure database operation)
func (db *Database) UpdatePoint(indexName string, id uint64, vector []float32) error {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("index not found")
	}

	// Convert Go slice to C array
	cVector := (*C.float)(unsafe.Pointer(&vector[0]))

	// Update point in HNSW index
	result := C.hnsw_update_point(index.CIndex, cVector, C.ulonglong(id))
	if result == -2 {
		return fmt.Errorf("point not found")
	}
	if result != 0 {
		return fmt.Errorf("failed to update point")
	}

	return nil
}

// UpdatePoints updates multiple points in an index (pure database operation)
func (db *Database) UpdatePoints(indexName string, points []*pb.Point) (int, error) {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return 0, fmt.Errorf("index not found")
	}

	numWorkers := runtime.NumCPU() / 4
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	var successCount atomic.Int32

	// Create work queue
	jobs := make(chan *pb.Point, len(points))

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for point := range jobs {
				cVector := (*C.float)(unsafe.Pointer(&point.Vector[0]))
				result := C.hnsw_update_point(index.CIndex, cVector, C.ulonglong(point.Id))
				if result == 0 {
					successCount.Add(1)
				}
			}
		}()
	}

	// Send jobs to workers
	for _, point := range points {
		jobs <- point
	}
	close(jobs)

	wg.Wait()

	return int(successCount.Load()), nil
}

// DeletePoint marks a single point as deleted in an index (soft delete, pure database operation)
func (db *Database) DeletePoint(indexName string, id uint64) error {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("index not found")
	}

	// Mark point as deleted in HNSW index
	result := C.hnsw_mark_deleted(index.CIndex, C.ulonglong(id))
	if result != 0 {
		return fmt.Errorf("failed to delete point")
	}

	return nil
}

// DeletePoints marks multiple points as deleted in an index (soft delete, pure database operation)
func (db *Database) DeletePoints(indexName string, ids []uint64) (int, error) {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return 0, fmt.Errorf("index not found")
	}

	numWorkers := runtime.NumCPU() / 4
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	var successCount atomic.Int32

	// Create work queue
	jobs := make(chan uint64, len(ids))

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				result := C.hnsw_mark_deleted(index.CIndex, C.ulonglong(id))
				if result == 0 {
					successCount.Add(1)
				}
			}
		}()
	}

	// Send jobs to workers
	for _, id := range ids {
		jobs <- id
	}
	close(jobs)

	wg.Wait()

	return int(successCount.Load()), nil
}

// SaveIndex saves an index to disk
func (db *Database) SaveIndex(indexName string) error {
	db.mutex.RLock()
	index, exists := db.indices[indexName]
	db.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("index %s not found", indexName)
	}

	// Ensure indices directory exists
	if err := os.MkdirAll(INDICES_DIR, 0755); err != nil {
		return fmt.Errorf("failed to create indices directory: %v", err)
	}

	// Save index data
	indexPath := filepath.Join(INDICES_DIR, indexName+INDEX_EXT)
	cIndexPath := C.CString(indexPath)
	defer C.free(unsafe.Pointer(cIndexPath))

	result := C.hnsw_save_index(index.CIndex, cIndexPath)
	if result != 0 {
		return fmt.Errorf("failed to save index data")
	}

	// Save metadata
	metadata := IndexMetadata{
		Name:                index.Name,
		Dimension:           index.Dimension,
		Space:               index.Space,
		MaxElements:         index.MaxElements,
		M:                   index.M,
		EfConstruct:         index.EfConstruct,
		AllowReplaceDeleted: index.AllowReplaceDeleted,
	}

	metadataPath := filepath.Join(INDICES_DIR, indexName+METADATA_EXT)
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %v", err)
	}
	defer metadataFile.Close()

	if err := json.NewEncoder(metadataFile).Encode(metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %v", err)
	}

	log.Printf("Successfully saved index %s", indexName)
	return nil
}

// LoadIndex loads an index from disk
func (db *Database) LoadIndex(indexName string) error {
	// Check if metadata file exists
	metadataPath := filepath.Join(INDICES_DIR, indexName+METADATA_EXT)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return fmt.Errorf("metadata file not found for index %s", indexName)
	}

	// Load metadata
	metadataFile, err := os.Open(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to open metadata file: %v", err)
	}
	defer metadataFile.Close()

	var metadata IndexMetadata
	if err := json.NewDecoder(metadataFile).Decode(&metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	// Check if index file exists
	indexPath := filepath.Join(INDICES_DIR, indexName+INDEX_EXT)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return fmt.Errorf("index file not found for index %s", indexName)
	}

	// Load index using C++ implementation
	cSpaceName := C.CString(metadata.Space)
	cIndexPath := C.CString(indexPath)
	defer C.free(unsafe.Pointer(cSpaceName))
	defer C.free(unsafe.Pointer(cIndexPath))

	cIndex := C.hnsw_load_index(
		cSpaceName,
		C.int(metadata.Dimension),
		cIndexPath,
		C.int(metadata.MaxElements),
	)

	if cIndex == nil {
		return fmt.Errorf("failed to load HNSW index from file")
	}

	// Create new index object
	index := &HNSWIndex{
		Name:                metadata.Name,
		Dimension:           metadata.Dimension,
		Space:               metadata.Space,
		MaxElements:         metadata.MaxElements,
		M:                   metadata.M,
		EfConstruct:         metadata.EfConstruct,
		AllowReplaceDeleted: metadata.AllowReplaceDeleted,
		CIndex:              cIndex,
	}

	db.mutex.Lock()
	db.indices[indexName] = index
	db.mutex.Unlock()

	log.Printf("Successfully loaded index %s", indexName)
	return nil
}

// SaveAllIndices saves all indices to disk
func (db *Database) SaveAllIndices() {
	db.mutex.RLock()
	indexNames := make([]string, 0, len(db.indices))
	for name := range db.indices {
		indexNames = append(indexNames, name)
	}
	db.mutex.RUnlock()

	for _, name := range indexNames {
		if err := db.SaveIndex(name); err != nil {
			log.Printf("Failed to save index %s: %v", name, err)
		}
	}
}

// LoadAllIndices loads all indices from disk
func (db *Database) LoadAllIndices() {
	// Check if indices directory exists
	if _, err := os.Stat(INDICES_DIR); os.IsNotExist(err) {
		log.Println("No indices directory found, starting with empty state")
		return
	}

	// Read directory contents
	files, err := os.ReadDir(INDICES_DIR)
	if err != nil {
		log.Printf("Failed to read indices directory: %v", err)
		return
	}

	// Find all metadata files and load corresponding indices
	for _, file := range files {
		if filepath.Ext(file.Name()) == METADATA_EXT {
			indexName := file.Name()[:len(file.Name())-len(METADATA_EXT)]
			if err := db.LoadIndex(indexName); err != nil {
				log.Printf("Failed to load index %s: %v", indexName, err)
			}
		}
	}
}
