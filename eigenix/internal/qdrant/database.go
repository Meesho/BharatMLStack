package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/qdrant/go-client/qdrant"
	"github.com/spf13/viper"
)

const (
	MaxWorkers = 32
)

// Database implements the Manager interface for Qdrant operations
type Database struct {
	client       *qdrant.Client
	centers      [][]float32 // Centroids stored as float32
	centersMutex sync.RWMutex
}

var (
	database Manager
	once     sync.Once
)

// Init initializes and returns the singleton database instance
func Init() Manager {
	if database == nil {
		once.Do(func() {
			client, err := qdrant.NewClient(&qdrant.Config{
				Host:   viper.GetString("QDRANT_HOST"),
				Port:   viper.GetInt("QDRANT_PORT"),
				UseTLS: false,
			})
			if err != nil {
				log.Fatalf("failed to create Qdrant client: %v", err)
			}
			database = &Database{
				client: client,
			}
		})
	}
	return database
}

// Close closes the Qdrant client connection
func (db *Database) Close() error {
	// Qdrant Go client doesn't require explicit close
	// But we can nil out the reference
	db.client = nil
	return nil
}

// LoadCentroids loads K-means centroids from a JSON file
func (db *Database) LoadCentroids(filepath string) error {
	db.centersMutex.Lock()
	defer db.centersMutex.Unlock()

	log.Printf("[qdrant] Loading centroids from %s...", filepath)

	// Load file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read centroids file: %w", err)
	}

	// Load as float64 from JSON
	var centersFloat64 [][]float64
	if err := json.Unmarshal(data, &centersFloat64); err != nil {
		return fmt.Errorf("failed to unmarshal centroids: %w", err)
	}

	if len(centersFloat64) == 0 {
		return fmt.Errorf("no centroids found in file")
	}

	rows := len(centersFloat64)
	cols := len(centersFloat64[0])

	// Convert to float32 and normalize
	db.centers = make([][]float32, rows)
	for i, center := range centersFloat64 {
		if len(center) != cols {
			return fmt.Errorf("inconsistent centroid dimensions at index %d", i)
		}

		// Convert to float32
		centerFloat32 := make([]float32, cols)
		for j, v := range center {
			centerFloat32[j] = float32(v)
		}

		// L2 normalize the center
		db.centers[i] = l2Normalize(centerFloat32)
	}

	log.Printf("[qdrant] Loaded %d centroids with %d dimensions (normalized)", rows, cols)

	return nil
}

// GetTopCentroids finds the top N closest centroids for a given embedding
func (db *Database) GetTopCentroids(embedding []float32, topN int) ([]int, error) {
	db.centersMutex.RLock()
	defer db.centersMutex.RUnlock()

	if db.centers == nil {
		return nil, fmt.Errorf("centroids not loaded")
	}

	// Normalize query vector
	q := l2Normalize(embedding)
	rows := len(db.centers)

	// Calculate similarities using dot product (cosine similarity for normalized vectors)
	sims := make([]struct {
		idx int
		sim float32
	}, rows)

	for i := 0; i < rows; i++ {
		sim := float32(0.0)
		for j := 0; j < len(q); j++ {
			sim += db.centers[i][j] * q[j]
		}
		sims[i] = struct {
			idx int
			sim float32
		}{i, sim}
	}

	// Sort by similarity descending
	sort.Slice(sims, func(i, j int) bool {
		return sims[i].sim > sims[j].sim
	})

	// Return top N centroid indices
	result := make([]int, topN)
	for i := 0; i < topN && i < len(sims); i++ {
		result[i] = sims[i].idx
	}

	return result, nil
}

// Search performs intelligent routing based on SearchParams
// - If UseIVF is true: routes to multiple centroid-based collections (_c0001, _c0002, etc.)
// - If UseIVF is false: queries single collection with index_name
func (db *Database) Search(ctx context.Context, req SearchRequest) ([]SearchResponse, error) {
	if db.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	if len(req.Vector) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}

	// Route based on IVF configuration
	if req.Params.UseIVF {
		// IVF mode: route to multiple centroid-based collections
		return db.searchIVF(ctx, req, req.Vector)
	}

	// Single collection mode: query the base collection directly
	return db.searchSingle(ctx, req, req.Vector)
}

// BatchSearch performs batch search across multiple query vectors in parallel
func (db *Database) BatchSearch(ctx context.Context, req BatchSearchRequest) ([][]SearchResponse, error) {
	fmt.Println("BatchSearch", req)
	if db.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	if len(req.Vectors) == 0 {
		return nil, fmt.Errorf("empty query vectors")
	}

	numVectors := len(req.Vectors)
	results := make([][]SearchResponse, numVectors)

	// Use worker pool for parallel processing
	type job struct {
		idx int
		vec []float32
	}

	type result struct {
		idx     int
		results []SearchResponse
		err     error
	}

	// Create job and result channels
	jobs := make(chan job, numVectors)
	resultsChan := make(chan result, numVectors)

	// Determine number of workers (use MaxWorkers or number of CPUs)
	numWorkers := MaxWorkers
	if numVectors < numWorkers {
		numWorkers = numVectors
	}

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				// Create a single search request
				searchReq := SearchRequest{
					IndexName: req.IndexName,
					Vector:    j.vec,
					Limit:     req.Limit,
					Params:    req.Params,
				}

				searchResults, err := db.Search(ctx, searchReq)
				if err != nil {
					resultsChan <- result{
						idx: j.idx,
						err: fmt.Errorf("search failed for vector %d: %w", j.idx, err),
					}
					continue
				}

				resultsChan <- result{
					idx:     j.idx,
					results: searchResults,
					err:     nil,
				}
			}
		}()
	}

	// Send jobs to workers
	go func() {
		for i, vec := range req.Vectors {
			if vec == nil || len(vec.Values) == 0 {
				resultsChan <- result{
					idx: i,
					err: fmt.Errorf("empty query vector at index %d", i),
				}
				continue
			}

			jobs <- job{
				idx: i,
				vec: vec.Values,
			}
		}
		close(jobs)
	}()

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for res := range resultsChan {
		if res.err != nil {
			return nil, res.err
		}
		results[res.idx] = res.results
	}

	return results, nil
}

// searchSingle performs search on a single collection
func (db *Database) searchSingle(ctx context.Context, req SearchRequest, queryVec []float32) ([]SearchResponse, error) {
	results, err := db.queryCollection(ctx, req.IndexName, queryVec, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("search failed for collection %s: %w", req.IndexName, err)
	}

	return results, nil
}

// searchIVF performs IVF-routed search across multiple centroid-based collections
func (db *Database) searchIVF(ctx context.Context, req SearchRequest, queryVec []float32) ([]SearchResponse, error) {
	// Get top centroids for routing
	topCentroids, err := db.GetTopCentroids(req.Vector, req.Params.CentroidCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get top centroids: %w", err)
	}

	// Build collection names from centroids (format: indexname_c0001, indexname_c0002, etc.)
	collections := make([]string, len(topCentroids))
	for i, cid := range topCentroids {
		collections[i] = fmt.Sprintf("%s_c%04d", req.IndexName, cid+1)
	}

	// Search across multiple collections concurrently
	allResults := db.searchMultipleCollections(ctx, collections, queryVec, req.Limit)

	// Dedupe and merge results
	merged := db.dedupeAndMerge(allResults, req.Limit)

	return merged, nil
}

// queryCollection performs a search on a single Qdrant collection
func (db *Database) queryCollection(ctx context.Context, collection string, queryVec []float32, limit uint64) ([]SearchResponse, error) {
	var results []SearchResponse

	queryPoints, err := db.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collection,
		Query: &qdrant.Query{
			Variant: &qdrant.Query_Nearest{
				Nearest: &qdrant.VectorInput{
					Variant: &qdrant.VectorInput_Dense{
						Dense: &qdrant.DenseVector{
							Data: queryVec,
						},
					},
				},
			},
		},
		Limit: &limit,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: false},
		},
		WithVectors: &qdrant.WithVectorsSelector{
			SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false},
		},
	})
	if err != nil {
		return nil, err
	}

	results = make([]SearchResponse, 0, len(queryPoints))
	for _, point := range queryPoints {
		results = append(results, SearchResponse{
			ID:    point.Id.GetNum(),
			Score: point.Score,
		})
	}

	return results, nil
}

// searchMultipleCollections searches across multiple collections concurrently
func (db *Database) searchMultipleCollections(ctx context.Context, collections []string, queryVec []float32, perCollectionLimit uint64) []SearchResponse {
	var wg sync.WaitGroup
	resultsChan := make(chan []SearchResponse, len(collections))

	// Limit concurrency
	sem := make(chan struct{}, MaxWorkers)

	for _, collection := range collections {
		wg.Add(1)
		go func(coll string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, err := db.queryCollection(ctx, coll, queryVec, perCollectionLimit)
			if err != nil {
				log.Printf("[warn] search failed for collection %s: %v", coll, err)
				return
			}
			resultsChan <- results
		}(collection)
	}

	wg.Wait()
	close(resultsChan)

	// Merge results from all collections
	allResults := []SearchResponse{}
	for results := range resultsChan {
		allResults = append(allResults, results...)
	}

	return allResults
}

// dedupeAndMerge deduplicates results by ID, keeping the best score, and limits to finalLimit
func (db *Database) dedupeAndMerge(results []SearchResponse, finalLimit uint64) []SearchResponse {
	// Dedupe by ID, keep best score
	byID := make(map[uint64]SearchResponse)
	for _, result := range results {
		if existing, ok := byID[result.ID]; !ok || result.Score > existing.Score {
			byID[result.ID] = result
		}
	}

	// Convert to slice
	merged := make([]SearchResponse, 0, len(byID))
	for _, result := range byID {
		merged = append(merged, result)
	}

	// Sort by score descending
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	// Limit to finalLimit
	if len(merged) > int(finalLimit) {
		merged = merged[:finalLimit]
	}

	return merged
}

// ==================== HELPER FUNCTIONS ====================

// l2Normalize normalizes a vector using L2 norm (float32 version)
func l2Normalize(vec []float32) []float32 {
	norm := float32(0.0)
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm))) + 1e-12

	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}
