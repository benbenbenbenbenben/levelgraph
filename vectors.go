// Copyright (c) 2024 LevelGraph Go Contributors
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package levelgraph

import (
	"context"
	"errors"
	"fmt"

	"github.com/benbenbenbenbenben/levelgraph/vector"
)

const defaultAsyncEmbedBufferSize = 100

var (
	// ErrVectorsDisabled is returned when vector operations are called without
	// a configured vector index.
	ErrVectorsDisabled = errors.New("levelgraph: vectors not enabled - use WithVectors option")

	// ErrEmbedderRequired is returned when auto-embedding requires an embedder
	// but none was configured.
	ErrEmbedderRequired = errors.New("levelgraph: embedder required for this operation")

	// ErrVectorDimensionMismatch is returned when loading a persisted vector
	// whose dimensions don't match the configured index dimensions.
	ErrVectorDimensionMismatch = errors.New("levelgraph: persisted vector dimensions do not match index dimensions")
)

// Key prefixes for vector storage in KVStore
var (
	vectorPrefix = []byte("vector::")
)

// VectorMatch represents a vector search result with graph context.
type VectorMatch struct {
	// ID is the vector identifier (e.g., "object:tennis").
	ID []byte
	// Score is the similarity score (higher is more similar).
	Score float32
	// Distance is the distance metric (lower is more similar).
	Distance float32
	// IDType indicates what kind of graph element this ID refers to.
	IDType vector.IDType
	// Parts contains the parsed ID components.
	Parts [][]byte
}

// SetVector associates a vector embedding with an ID.
// The ID can be created using vector.MakeID to associate vectors with
// graph elements (subjects, objects, predicates, triples, or facets).
//
// Example:
//
//	// Associate a vector with an object value
//	id := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
//	db.SetVector(ctx, id, tennisEmbedding)
//
//	// Associate a vector with a custom ID
//	db.SetVector(ctx, []byte("doc:123"), docEmbedding)
func (db *DB) SetVector(ctx context.Context, id []byte, vec []float32) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	if db.options.VectorIndex == nil {
		return ErrVectorsDisabled
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	// Add to vector index
	if err := db.options.VectorIndex.Add(id, vec); err != nil {
		return fmt.Errorf("levelgraph: set vector: %w", err)
	}

	// Persist to KVStore for durability
	key := makeVectorKey(id)
	value := vector.VectorToBytes(vec)
	if err := db.store.Put(key, value, nil); err != nil {
		// Try to rollback from index
		db.options.VectorIndex.Delete(id)
		return fmt.Errorf("levelgraph: persist vector: %w", err)
	}

	if db.options.Logger != nil {
		db.options.Logger.Debug("set vector", "id", string(id), "dims", len(vec))
	}

	return nil
}

// GetVector retrieves a vector embedding by ID.
func (db *DB) GetVector(ctx context.Context, id []byte) ([]float32, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	if db.options.VectorIndex == nil {
		return nil, ErrVectorsDisabled
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	return db.options.VectorIndex.Get(id)
}

// DeleteVector removes a vector embedding by ID.
func (db *DB) DeleteVector(ctx context.Context, id []byte) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	if db.options.VectorIndex == nil {
		return ErrVectorsDisabled
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	// Delete from index
	if err := db.options.VectorIndex.Delete(id); err != nil {
		return fmt.Errorf("levelgraph: delete vector: %w", err)
	}

	// Delete from KVStore
	key := makeVectorKey(id)
	if err := db.store.Delete(key, nil); err != nil {
		return fmt.Errorf("levelgraph: delete persisted vector: %w", err)
	}

	if db.options.Logger != nil {
		db.options.Logger.Debug("delete vector", "id", string(id))
	}

	return nil
}

// SearchVectors finds the k most similar vectors to the query.
// Results are sorted by similarity (highest first).
//
// Example:
//
//	// Find objects similar to "racket sports"
//	queryVec, _ := embedder.Embed("racket sports")
//	results, _ := db.SearchVectors(ctx, queryVec, 10)
//	for _, r := range results {
//	    fmt.Printf("%s: %.3f\n", r.Parts[0], r.Score)
//	}
func (db *DB) SearchVectors(ctx context.Context, query []float32, k int) ([]VectorMatch, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	if db.options.VectorIndex == nil {
		return nil, ErrVectorsDisabled
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	matches, err := db.options.VectorIndex.Search(query, k)
	if err != nil {
		return nil, fmt.Errorf("levelgraph: search vectors: %w", err)
	}

	results := make([]VectorMatch, len(matches))
	for i, m := range matches {
		idType, parts := vector.ParseID(m.ID)
		results[i] = VectorMatch{
			ID:       m.ID,
			Score:    m.Score,
			Distance: m.Distance,
			IDType:   idType,
			Parts:    parts,
		}
	}

	if db.options.Logger != nil {
		db.options.Logger.Debug("search vectors", "k", k, "results", len(results))
	}

	return results, nil
}

// SearchVectorsByText searches for similar vectors using text input.
// Requires an Embedder to be configured (via WithAutoEmbed).
//
// Example:
//
//	results, _ := db.SearchVectorsByText(ctx, "racket sports", 10)
func (db *DB) SearchVectorsByText(ctx context.Context, text string, k int) ([]VectorMatch, error) {
	db.mu.RLock()

	if db.closed {
		db.mu.RUnlock()
		return nil, fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	if db.options.VectorIndex == nil {
		db.mu.RUnlock()
		return nil, ErrVectorsDisabled
	}

	if db.options.Embedder == nil {
		db.mu.RUnlock()
		return nil, ErrEmbedderRequired
	}

	select {
	case <-ctx.Done():
		db.mu.RUnlock()
		return nil, fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	// Embed the query text
	queryVec, err := db.options.Embedder.Embed(text)
	if err != nil {
		db.mu.RUnlock()
		return nil, fmt.Errorf("levelgraph: embed query: %w", err)
	}

	// Release our lock before calling SearchVectors, which will acquire its own lock.
	// This avoids potential deadlock and double-unlock issues.
	db.mu.RUnlock()

	return db.SearchVectors(ctx, queryVec, k)
}

// EmbedAndSetVector embeds text and stores the resulting vector.
// Requires an Embedder to be configured.
//
// Example:
//
//	id := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
//	db.EmbedAndSetVector(ctx, id, "tennis is a racket sport")
func (db *DB) EmbedAndSetVector(ctx context.Context, id []byte, text string) error {
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	if db.options.VectorIndex == nil {
		db.mu.RUnlock()
		return ErrVectorsDisabled
	}

	if db.options.Embedder == nil {
		db.mu.RUnlock()
		return ErrEmbedderRequired
	}
	db.mu.RUnlock()

	select {
	case <-ctx.Done():
		return fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	// Embed the text
	vec, err := db.options.Embedder.Embed(text)
	if err != nil {
		return fmt.Errorf("levelgraph: embed text: %w", err)
	}

	return db.SetVector(ctx, id, vec)
}

// VectorCount returns the number of vectors in the index.
func (db *DB) VectorCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.options.VectorIndex == nil {
		return 0
	}

	return db.options.VectorIndex.Len()
}

// VectorDimensions returns the dimensionality of the vector index.
// Returns 0 if vectors are not enabled.
func (db *DB) VectorDimensions() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.options.VectorIndex == nil {
		return 0
	}

	return db.options.VectorIndex.Dimensions()
}

// VectorsEnabled returns true if vector operations are available.
func (db *DB) VectorsEnabled() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.options.VectorIndex != nil
}

// LoadVectors loads all persisted vectors from KVStore into the index.
// This should be called after opening a database with vectors enabled
// to restore the index state.
func (db *DB) LoadVectors(ctx context.Context) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	if db.options.VectorIndex == nil {
		return ErrVectorsDisabled
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	// Iterate over all vector keys
	start := vectorPrefix
	end := append([]byte{}, vectorPrefix...)
	end[len(end)-1]++ // Increment last byte for range end

	iter := db.store.NewIterator(&Range{Start: start, Limit: end}, nil)
	defer iter.Release()

	expectedDims := db.options.VectorIndex.Dimensions()
	count := 0
	for iter.Next() {
		select {
		case <-ctx.Done():
			return fmt.Errorf("levelgraph: %w", ctx.Err())
		default:
		}

		// Extract ID from key
		key := iter.Key()
		id := key[len(vectorPrefix):]

		// Parse vector from value
		vec := vector.BytesToVector(iter.Value())
		if vec == nil {
			continue
		}

		// Validate dimensions match the index
		if len(vec) != expectedDims {
			return fmt.Errorf("%w: vector %q has %d dimensions, index expects %d",
				ErrVectorDimensionMismatch, id, len(vec), expectedDims)
		}

		// Add to index
		if err := db.options.VectorIndex.Add(id, vec); err != nil {
			return fmt.Errorf("levelgraph: load vector %s: %w", id, err)
		}
		count++
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("levelgraph: iterate vectors: %w", err)
	}

	if db.options.Logger != nil {
		db.options.Logger.Info("loaded vectors", "count", count)
	}

	return nil
}

// makeVectorKey creates a storage key for a vector ID.
func makeVectorKey(id []byte) []byte {
	key := make([]byte, len(vectorPrefix)+len(id))
	copy(key, vectorPrefix)
	copy(key[len(vectorPrefix):], id)
	return key
}

// SetSubjectVector is a convenience method to set a vector for a subject value.
func (db *DB) SetSubjectVector(ctx context.Context, subject []byte, vec []float32) error {
	id := vector.MakeID(vector.IDTypeSubject, subject)
	return db.SetVector(ctx, id, vec)
}

// SetObjectVector is a convenience method to set a vector for an object value.
func (db *DB) SetObjectVector(ctx context.Context, object []byte, vec []float32) error {
	id := vector.MakeID(vector.IDTypeObject, object)
	return db.SetVector(ctx, id, vec)
}

// SetTripleVector is a convenience method to set a vector for a triple.
func (db *DB) SetTripleVector(ctx context.Context, triple *Triple, vec []float32) error {
	id := vector.MakeID(vector.IDTypeTriple, triple.Subject, triple.Predicate, triple.Object)
	return db.SetVector(ctx, id, vec)
}

// SearchSimilarObjects searches for objects similar to a query vector.
// Only returns matches with IDTypeObject.
func (db *DB) SearchSimilarObjects(ctx context.Context, query []float32, k int) ([]VectorMatch, error) {
	results, err := db.SearchVectors(ctx, query, k*2) // Fetch more to filter
	if err != nil {
		return nil, err
	}

	filtered := make([]VectorMatch, 0, k)
	for _, r := range results {
		if r.IDType == vector.IDTypeObject {
			filtered = append(filtered, r)
			if len(filtered) >= k {
				break
			}
		}
	}

	return filtered, nil
}

// SearchSimilarSubjects searches for subjects similar to a query vector.
// Only returns matches with IDTypeSubject.
func (db *DB) SearchSimilarSubjects(ctx context.Context, query []float32, k int) ([]VectorMatch, error) {
	results, err := db.SearchVectors(ctx, query, k*2) // Fetch more to filter
	if err != nil {
		return nil, err
	}

	filtered := make([]VectorMatch, 0, k)
	for _, r := range results {
		if r.IDType == vector.IDTypeSubject {
			filtered = append(filtered, r)
			if len(filtered) >= k {
				break
			}
		}
	}

	return filtered, nil
}

// autoEmbedTriples generates and stores vector embeddings for triple components
// based on the configured AutoEmbedTargets. This is called automatically during Put()
// when both an Embedder and VectorIndex are configured.
//
// If AsyncAutoEmbed is enabled, this queues the work for background processing.
// Otherwise, it processes synchronously.
func (db *DB) autoEmbedTriples(ctx context.Context, triples []*Triple) error {
	// If async embedding is enabled, queue the work
	if db.embedStarted {
		// Make a copy of the triples slice to avoid races
		triplesCopy := make([]*Triple, len(triples))
		copy(triplesCopy, triples)

		db.embedWg.Add(1)
		select {
		case db.embedQueue <- triplesCopy:
			// Successfully queued
			return nil
		case <-ctx.Done():
			db.embedWg.Done()
			return ctx.Err()
		}
	}

	// Synchronous embedding
	return db.doAutoEmbedTriples(ctx, triples)
}

// doAutoEmbedTriples performs the actual embedding work.
// This is called either synchronously from autoEmbedTriples or from the background worker.
func (db *DB) doAutoEmbedTriples(ctx context.Context, triples []*Triple) error {
	// Collect unique values to embed by type
	subjects := make(map[string][]byte)
	predicates := make(map[string][]byte)
	objects := make(map[string][]byte)

	targets := db.options.AutoEmbedTargets

	for _, triple := range triples {
		if targets&AutoEmbedSubjects != 0 {
			key := string(triple.Subject)
			if _, exists := subjects[key]; !exists {
				subjects[key] = triple.Subject
			}
		}
		if targets&AutoEmbedPredicates != 0 {
			key := string(triple.Predicate)
			if _, exists := predicates[key]; !exists {
				predicates[key] = triple.Predicate
			}
		}
		if targets&AutoEmbedObjects != 0 {
			key := string(triple.Object)
			if _, exists := objects[key]; !exists {
				objects[key] = triple.Object
			}
		}
	}

	// Batch embed all texts
	var texts []string
	var ids [][]byte

	for _, val := range subjects {
		// Skip if vector already exists
		id := vector.MakeID(vector.IDTypeSubject, val)
		if _, err := db.options.VectorIndex.Get(id); err == nil {
			continue
		}
		texts = append(texts, string(val))
		ids = append(ids, id)
	}
	for _, val := range predicates {
		id := vector.MakeID(vector.IDTypePredicate, val)
		if _, err := db.options.VectorIndex.Get(id); err == nil {
			continue
		}
		texts = append(texts, string(val))
		ids = append(ids, id)
	}
	for _, val := range objects {
		id := vector.MakeID(vector.IDTypeObject, val)
		if _, err := db.options.VectorIndex.Get(id); err == nil {
			continue
		}
		texts = append(texts, string(val))
		ids = append(ids, id)
	}

	if len(texts) == 0 {
		return nil // Nothing new to embed
	}

	// Embed all texts
	embeddings, err := db.options.Embedder.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("embed batch: %w", err)
	}

	// Store vectors (we already hold the read lock from Put, need to release/reacquire)
	// Note: We're inside Put which holds db.mu.RLock(), but SetVector also tries to RLock.
	// Go's RWMutex allows multiple concurrent RLocks, so this is safe.
	for i, id := range ids {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Add to index
		if err := db.options.VectorIndex.Add(id, embeddings[i]); err != nil {
			return fmt.Errorf("add vector: %w", err)
		}

		// Persist to KVStore
		key := makeVectorKey(id)
		value := vector.VectorToBytes(embeddings[i])
		if err := db.store.Put(key, value, nil); err != nil {
			// Try to rollback from index
			db.options.VectorIndex.Delete(id)
			return fmt.Errorf("persist vector: %w", err)
		}
	}

	if db.options.Logger != nil {
		db.options.Logger.Debug("auto-embedded", "count", len(ids))
	}

	return nil
}

// startEmbedWorker starts the background embedding worker if async embedding is enabled.
func (db *DB) startEmbedWorker() {
	if !db.options.AsyncAutoEmbed {
		return
	}

	bufSize := db.options.AsyncEmbedBufferSize
	if bufSize <= 0 {
		bufSize = defaultAsyncEmbedBufferSize
	}

	db.embedQueue = make(chan []*Triple, bufSize)
	db.embedDone = make(chan struct{})
	db.embedStarted = true

	go db.embedWorker()
}

// stopEmbedWorker stops the background embedding worker and waits for it to finish.
func (db *DB) stopEmbedWorker() {
	if !db.embedStarted {
		return
	}

	// Close the queue to signal worker to stop
	close(db.embedQueue)

	// Wait for worker to finish processing all items
	<-db.embedDone

	if db.options.Logger != nil {
		db.options.Logger.Debug("embed worker stopped")
	}
}

// embedWorker is the background goroutine that processes embedding requests.
func (db *DB) embedWorker() {
	defer close(db.embedDone)

	ctx := context.Background()

	for triples := range db.embedQueue {
		// Process the embedding request
		if err := db.doAutoEmbedTriples(ctx, triples); err != nil {
			if db.options.Logger != nil {
				db.options.Logger.Warn("async auto-embed failed", "error", err)
			}
		}
		db.embedWg.Done()
	}

	if db.options.Logger != nil {
		db.options.Logger.Debug("embed worker finished")
	}
}

// WaitForEmbeddings blocks until all pending async embedding operations are complete.
// Returns immediately if async embedding is not enabled.
// Returns an error if the context is cancelled before all embeddings complete.
//
// Use this method after a batch of Put operations to ensure all vectors are
// indexed before performing searches:
//
//	// Add triples with async embedding
//	for _, triple := range triples {
//	    db.Put(ctx, triple)
//	}
//
//	// Wait for all embeddings to complete before searching
//	if err := db.WaitForEmbeddings(ctx); err != nil {
//	    log.Printf("embedding error: %v", err)
//	}
//
//	// Now search will include all vectors
//	results, _ := db.SearchVectorsByText(ctx, "query", 10)
func (db *DB) WaitForEmbeddings(ctx context.Context) error {
	if !db.embedStarted {
		return nil
	}

	done := make(chan struct{})
	go func() {
		db.embedWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("levelgraph: wait for embeddings: %w", ctx.Err())
	}
}

// PendingEmbeddings returns the number of pending async embedding operations.
// Returns 0 if async embedding is not enabled.
func (db *DB) PendingEmbeddings() int {
	if !db.embedStarted || db.embedQueue == nil {
		return 0
	}
	return len(db.embedQueue)
}
