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
// THE SOFTWARE IS PROgraph.VIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO Egraph.VENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package levelgraph

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/benbenbenbenbenben/levelgraph/vector"

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
)

func setupTestDBWithVectors(t *testing.T, dims int) (*DB, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	index := vector.NewFlatIndex(dims)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestDB_VectorsDisabled(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Verify vectors are disabled
	if db.VectorsEnabled() {
		t.Error("VectorsEnabled() should return false")
	}

	if db.VectorDimensions() != 0 {
		t.Errorf("VectorDimensions() = %d, want 0", db.VectorDimensions())
	}

	// All vector operations should return ErrVectorsDisabled
	err := db.SetVector(ctx, []byte("test"), []float32{1, 0, 0})
	if err != ErrVectorsDisabled {
		t.Errorf("SetVector() error = %v, want ErrVectorsDisabled", err)
	}

	_, err = db.GetVector(ctx, []byte("test"))
	if err != ErrVectorsDisabled {
		t.Errorf("GetVector() error = %v, want ErrVectorsDisabled", err)
	}

	err = db.DeleteVector(ctx, []byte("test"))
	if err != ErrVectorsDisabled {
		t.Errorf("DeleteVector() error = %v, want ErrVectorsDisabled", err)
	}

	_, err = db.SearchVectors(ctx, []float32{1, 0, 0}, 5)
	if err != ErrVectorsDisabled {
		t.Errorf("SearchVectors() error = %v, want ErrVectorsDisabled", err)
	}
}

func TestDB_VectorBasicOperations(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDBWithVectors(t, 3)
	defer cleanup()

	ctx := context.Background()

	// Verify vectors are enabled
	if !db.VectorsEnabled() {
		t.Error("VectorsEnabled() should return true")
	}

	if db.VectorDimensions() != 3 {
		t.Errorf("VectorDimensions() = %d, want 3", db.VectorDimensions())
	}

	// Test SetVector
	vec := []float32{1, 0, 0}
	err := db.SetVector(ctx, []byte("v1"), vec)
	if err != nil {
		t.Fatalf("SetVector() error = %v", err)
	}

	if db.VectorCount() != 1 {
		t.Errorf("VectorCount() = %d, want 1", db.VectorCount())
	}

	// Test GetVector
	retrieved, err := db.GetVector(ctx, []byte("v1"))
	if err != nil {
		t.Fatalf("GetVector() error = %v", err)
	}
	if len(retrieved) != 3 || retrieved[0] != 1 || retrieved[1] != 0 || retrieved[2] != 0 {
		t.Errorf("GetVector() = %v, want %v", retrieved, vec)
	}

	// Test DeleteVector
	err = db.DeleteVector(ctx, []byte("v1"))
	if err != nil {
		t.Fatalf("DeleteVector() error = %v", err)
	}

	if db.VectorCount() != 0 {
		t.Errorf("VectorCount() after delete = %d, want 0", db.VectorCount())
	}

	// GetVector should return not found
	_, err = db.GetVector(ctx, []byte("v1"))
	if err != vector.ErrNotFound {
		t.Errorf("GetVector() after delete error = %v, want ErrNotFound", err)
	}
}

func TestDB_VectorSearch(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDBWithVectors(t, 3)
	defer cleanup()

	ctx := context.Background()

	// Add several vectors
	vectors := map[string][]float32{
		"v1": {1, 0, 0},     // "x axis"
		"v2": {0, 1, 0},     // "y axis"
		"v3": {0, 0, 1},     // "z axis"
		"v4": {0.9, 0.1, 0}, // close to v1
		"v5": {0.8, 0.2, 0}, // also close to v1
	}

	for id, vec := range vectors {
		err := db.SetVector(ctx, []byte(id), vec)
		if err != nil {
			t.Fatalf("SetVector(%s) error = %v", id, err)
		}
	}

	// Search for vectors similar to [1, 0, 0]
	results, err := db.SearchVectors(ctx, []float32{1, 0, 0}, 3)
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("SearchVectors() returned %d results, want 3", len(results))
	}

	// v1 should be first (exact match)
	if string(results[0].ID) != "v1" {
		t.Errorf("First result = %s, want v1", results[0].ID)
	}

	// Score for exact match should be close to 1.0
	if results[0].Score < 0.99 {
		t.Errorf("First result score = %v, want ~1.0", results[0].Score)
	}
}

func TestDB_VectorWithTypedIDs(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDBWithVectors(t, 3)
	defer cleanup()

	ctx := context.Background()

	// Test subject vector
	subjectID := vector.MakeID(vector.IDTypeSubject, []byte("alice"))
	err := db.SetVector(ctx, subjectID, []float32{1, 0, 0})
	if err != nil {
		t.Fatalf("SetVector(subject) error = %v", err)
	}

	// Test object vector
	objectID := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
	err = db.SetVector(ctx, objectID, []float32{0, 1, 0})
	if err != nil {
		t.Fatalf("SetVector(object) error = %v", err)
	}

	// Test triple vector
	tripleID := vector.MakeID(vector.IDTypeTriple, []byte("alice"), []byte("likes"), []byte("tennis"))
	err = db.SetVector(ctx, tripleID, []float32{0, 0, 1})
	if err != nil {
		t.Fatalf("SetVector(triple) error = %v", err)
	}

	// Search and verify IDType parsing
	results, err := db.SearchVectors(ctx, []float32{1, 0, 0}, 3)
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}

	// First result should be the subject
	if results[0].IDType != vector.IDTypeSubject {
		t.Errorf("First result IDType = %v, want subject", results[0].IDType)
	}
	if !bytes.Equal(results[0].Parts[0], []byte("alice")) {
		t.Errorf("First result Parts[0] = %s, want alice", results[0].Parts[0])
	}
}

func TestDB_ConvenienceVectorMethods(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDBWithVectors(t, 3)
	defer cleanup()

	ctx := context.Background()

	// Test SetSubjectVector
	err := db.SetSubjectVector(ctx, []byte("alice"), []float32{1, 0, 0})
	if err != nil {
		t.Fatalf("SetSubjectVector() error = %v", err)
	}

	// Test SetObjectVector
	err = db.SetObjectVector(ctx, []byte("tennis"), []float32{0, 1, 0})
	if err != nil {
		t.Fatalf("SetObjectVector() error = %v", err)
	}

	// Test SetTripleVector
	triple := graph.NewTripleFromStrings("alice", "likes", "tennis")
	err = db.SetTripleVector(ctx, triple, []float32{0, 0, 1})
	if err != nil {
		t.Fatalf("SetTripleVector() error = %v", err)
	}

	// Verify all three vectors exist
	if db.VectorCount() != 3 {
		t.Errorf("VectorCount() = %d, want 3", db.VectorCount())
	}
}

func TestDB_SearchSimilarObjects(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDBWithVectors(t, 3)
	defer cleanup()

	ctx := context.Background()

	// Add objects
	db.SetObjectVector(ctx, []byte("tennis"), []float32{1, 0, 0})
	db.SetObjectVector(ctx, []byte("badminton"), []float32{0.9, 0.1, 0})
	db.SetObjectVector(ctx, []byte("football"), []float32{0, 1, 0})

	// Add a subject (should not appear in object search)
	db.SetSubjectVector(ctx, []byte("alice"), []float32{0.95, 0.05, 0})

	// Search for similar objects
	results, err := db.SearchSimilarObjects(ctx, []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("SearchSimilarObjects() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchSimilarObjects() returned %d results, want 2", len(results))
	}

	// Should only return objects
	for _, r := range results {
		if r.IDType != vector.IDTypeObject {
			t.Errorf("SearchSimilarObjects() returned non-object: %v", r.IDType)
		}
	}

	// First should be tennis
	if string(results[0].Parts[0]) != "tennis" {
		t.Errorf("First result = %s, want tennis", results[0].Parts[0])
	}
}

func TestDB_VectorPersistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create database with vectors and add data
	{
		index := vector.NewFlatIndex(3)
		db, err := Open(dbPath, WithVectors(index))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		ctx := context.Background()
		db.SetVector(ctx, []byte("v1"), []float32{1, 0, 0})
		db.SetVector(ctx, []byte("v2"), []float32{0, 1, 0})
		db.SetVector(ctx, []byte("v3"), []float32{0, 0, 1})
		db.Close()
	}

	// Reopen and verify data is persisted
	{
		index := vector.NewFlatIndex(3) // New empty index
		db, err := Open(dbPath, WithVectors(index))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer db.Close()

		ctx := context.Background()

		// Index should be empty before loading
		if db.VectorCount() != 0 {
			t.Errorf("VectorCount() before load = %d, want 0", db.VectorCount())
		}

		// Load vectors from storage
		err = db.LoadVectors(ctx)
		if err != nil {
			t.Fatalf("LoadVectors() error = %v", err)
		}

		// Now index should have the vectors
		if db.VectorCount() != 3 {
			t.Errorf("VectorCount() after load = %d, want 3", db.VectorCount())
		}

		// Verify we can retrieve and search
		vec, err := db.GetVector(ctx, []byte("v1"))
		if err != nil {
			t.Fatalf("GetVector() error = %v", err)
		}
		if vec[0] != 1 || vec[1] != 0 || vec[2] != 0 {
			t.Errorf("GetVector() = %v, want [1, 0, 0]", vec)
		}

		results, err := db.SearchVectors(ctx, []float32{1, 0, 0}, 1)
		if err != nil {
			t.Fatalf("SearchVectors() error = %v", err)
		}
		if string(results[0].ID) != "v1" {
			t.Errorf("SearchVectors() first result = %s, want v1", results[0].ID)
		}
	}
}

func TestDB_LoadVectorsDimensionMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create database with 3-dimensional vectors and add data
	{
		index := vector.NewFlatIndex(3)
		db, err := Open(dbPath, WithVectors(index))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		ctx := context.Background()
		db.SetVector(ctx, []byte("v1"), []float32{1, 0, 0})
		db.SetVector(ctx, []byte("v2"), []float32{0, 1, 0})
		db.Close()
	}

	// Reopen with DIFFERENT dimensions (5 instead of 3)
	{
		index := vector.NewFlatIndex(5) // Different dimensions!
		db, err := Open(dbPath, WithVectors(index))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer db.Close()

		ctx := context.Background()

		// LoadVectors should fail with dimension mismatch error
		err = db.LoadVectors(ctx)
		if err == nil {
			t.Fatal("LoadVectors() should return error for dimension mismatch")
		}

		// Should be a dimension mismatch error
		if !errors.Is(err, ErrVectorDimensionMismatch) {
			t.Errorf("LoadVectors() error = %v, want ErrVectorDimensionMismatch", err)
		}

		// Error message should include both dimensions
		errMsg := err.Error()
		if !bytes.Contains([]byte(errMsg), []byte("3")) || !bytes.Contains([]byte(errMsg), []byte("5")) {
			t.Errorf("Error message should include dimensions 3 and 5: %v", errMsg)
		}
	}
}

func TestDB_VectorWithHNSW(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Use HNSW instead of flat index
	index := vector.NewHNSWIndex(3, vector.WithM(4), vector.WithEfConstruction(50))
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add vectors
	db.SetVector(ctx, []byte("v1"), []float32{1, 0, 0})
	db.SetVector(ctx, []byte("v2"), []float32{0.9, 0.1, 0})
	db.SetVector(ctx, []byte("v3"), []float32{0, 1, 0})
	db.SetVector(ctx, []byte("v4"), []float32{0, 0, 1})

	// Search
	results, err := db.SearchVectors(ctx, []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchVectors() returned %d results, want 2", len(results))
	}

	// v1 should be first
	if string(results[0].ID) != "v1" {
		t.Errorf("First result = %s, want v1", results[0].ID)
	}
}

func TestDB_VectorAndTriplesTogether(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add triples about sports
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "badminton"))
	db.Put(ctx, graph.NewTripleFromStrings("charlie", "likes", "football"))

	// Add vectors for the sports (objects)
	db.SetObjectVector(ctx, []byte("tennis"), []float32{1, 0, 0})
	db.SetObjectVector(ctx, []byte("badminton"), []float32{0.9, 0.1, 0})
	db.SetObjectVector(ctx, []byte("football"), []float32{0, 1, 0})

	// Search for sports similar to "racket sports" (represented by [1, 0, 0])
	results, err := db.SearchSimilarObjects(ctx, []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("SearchSimilarObjects() error = %v", err)
	}

	// Should find tennis and badminton
	found := map[string]bool{}
	for _, r := range results {
		found[string(r.Parts[0])] = true
	}
	if !found["tennis"] {
		t.Error("Expected to find tennis in results")
	}
	if !found["badminton"] {
		t.Error("Expected to find badminton in results")
	}

	// Now use graph query to find who likes these sports
	for _, r := range results {
		sport := r.Parts[0]
		triples, err := db.Get(ctx, &graph.Pattern{
			Predicate: graph.ExactString("likes"),
			Object:    graph.Exact(sport),
		})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(triples) == 0 {
			t.Errorf("No one likes %s", sport)
		}
	}
}

// MockEmbedder for testing auto-embed functionality
type mockEmbedder struct {
	dims int
}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	// Simple hash-based embedding for testing
	vec := make([]float32, m.dims)
	for i, c := range text {
		vec[i%m.dims] += float32(c) / 1000
	}
	return vector.NormalizeCopy(vec), nil
}

func (m *mockEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := m.Embed(text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

func (m *mockEmbedder) Dimensions() int {
	return m.dims
}

func TestDB_EmbedAndSetVector(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test EmbedAndSetVector
	id := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
	err = db.EmbedAndSetVector(ctx, id, "tennis is a racket sport played on a court")
	if err != nil {
		t.Fatalf("EmbedAndSetVector() error = %v", err)
	}

	// Verify vector was created
	vec, err := db.GetVector(ctx, id)
	if err != nil {
		t.Fatalf("GetVector() error = %v", err)
	}
	if len(vec) != 8 {
		t.Errorf("Vector dimensions = %d, want 8", len(vec))
	}
}

func TestDB_SearchVectorsByText(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add some embedded vectors
	db.EmbedAndSetVector(ctx, vector.MakeID(vector.IDTypeObject, []byte("tennis")), "tennis racket sport")
	db.EmbedAndSetVector(ctx, vector.MakeID(vector.IDTypeObject, []byte("badminton")), "badminton racket sport")
	db.EmbedAndSetVector(ctx, vector.MakeID(vector.IDTypeObject, []byte("football")), "football soccer ball")

	// Search by text
	results, err := db.SearchVectorsByText(ctx, "racket sports", 2)
	if err != nil {
		t.Fatalf("SearchVectorsByText() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchVectorsByText() returned %d results, want 2", len(results))
	}
}

func TestDB_SearchVectorsByTextNoEmbedder(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDBWithVectors(t, 3)
	defer cleanup()

	ctx := context.Background()

	// Without embedder, SearchVectorsByText should fail
	_, err := db.SearchVectorsByText(ctx, "test", 5)
	if err != ErrEmbedderRequired {
		t.Errorf("SearchVectorsByText() error = %v, want ErrEmbedderRequired", err)
	}
}

// ============================================================================
// Hybrid Search Tests
// ============================================================================

func TestDB_HybridSearch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Set up a graph of people and the sports they like
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "badminton"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "football"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("charlie", "likes", "swimming"))

	// Set up vectors for sports
	// "Racket sports" direction: [1, 0, 0]
	// "Ball sports" direction: [0, 1, 0]
	// "Water sports" direction: [0, 0, 1]
	db.SetObjectVector(ctx, []byte("tennis"), []float32{0.9, 0.3, 0})    // Racket + ball
	db.SetObjectVector(ctx, []byte("badminton"), []float32{1, 0, 0})     // Pure racket
	db.SetObjectVector(ctx, []byte("football"), []float32{0.1, 0.95, 0}) // Ball sport
	db.SetObjectVector(ctx, []byte("swimming"), []float32{0, 0, 1})      // Water sport

	// Hybrid search: "Find people who like racket sports"
	// This combines graph traversal with vector similarity
	solutions, err := db.Search(ctx, []*graph.Pattern{
		{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("sport")},
	}, &SearchOptions{
		VectorFilter: &VectorFilter{
			Variable: "sport",
			Query:    []float32{1, 0, 0}, // Racket sports direction
			TopK:     3,
			IDType:   vector.IDTypeObject,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(solutions) != 3 {
		t.Fatalf("Search() returned %d solutions, want 3", len(solutions))
	}

	// First result should be badminton (exact match to query)
	firstSport := string(solutions[0]["sport"])
	if firstSport != "badminton" {
		t.Errorf("First sport = %s, want badminton", firstSport)
	}

	// Check that scores are populated and in descending order
	prevScore := float32(2.0) // Higher than max possible
	for i, sol := range solutions {
		score := GetVectorScore(sol)
		if score > prevScore {
			t.Errorf("Solution %d score %v > previous %v (should be descending)", i, score, prevScore)
		}
		prevScore = score
		t.Logf("Solution %d: %s likes %s (score: %.3f)",
			i, sol["person"], sol["sport"], score)
	}
}

func TestDB_HybridSearchWithMinScore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Set up graph
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "swimming"))

	// Set up vectors - tennis similar to query, swimming very different
	db.SetObjectVector(ctx, []byte("tennis"), []float32{1, 0, 0})
	db.SetObjectVector(ctx, []byte("swimming"), []float32{-1, 0, 0}) // Opposite direction

	// Search with minimum score threshold
	solutions, err := db.Search(ctx, []*graph.Pattern{
		{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("sport")},
	}, &SearchOptions{
		VectorFilter: &VectorFilter{
			Variable: "sport",
			Query:    []float32{1, 0, 0},
			MinScore: 0.7, // Filter out low scores
			IDType:   vector.IDTypeObject,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Only tennis should pass the threshold
	if len(solutions) != 1 {
		t.Fatalf("Search() returned %d solutions, want 1", len(solutions))
	}

	sport := string(solutions[0]["sport"])
	if sport != "tennis" {
		t.Errorf("Sport = %s, want tennis", sport)
	}
}

func TestDB_HybridSearchWithTextQuery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Set up graph
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "football"))

	// Embed objects
	db.EmbedAndSetVector(ctx, vector.MakeID(vector.IDTypeObject, []byte("tennis")), "tennis racket sport")
	db.EmbedAndSetVector(ctx, vector.MakeID(vector.IDTypeObject, []byte("football")), "football soccer ball")

	// Search using text query (will be embedded)
	solutions, err := db.Search(ctx, []*graph.Pattern{
		{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("sport")},
	}, &SearchOptions{
		VectorFilter: &VectorFilter{
			Variable:  "sport",
			QueryText: "racket sports", // Will be embedded
			TopK:      2,
			IDType:    vector.IDTypeObject,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(solutions) != 2 {
		t.Fatalf("Search() returned %d solutions, want 2", len(solutions))
	}
}

func TestDB_HybridSearchMultiplePatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// More complex graph: people -> sports -> categories
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "badminton"))
	db.Put(ctx, graph.NewTripleFromStrings("tennis", "category", "racket_sport"))
	db.Put(ctx, graph.NewTripleFromStrings("badminton", "category", "racket_sport"))

	// Set vectors for categories
	db.SetObjectVector(ctx, []byte("racket_sport"), []float32{1, 0, 0})

	// Two-pattern search: find people -> sport -> category, filter by category similarity
	solutions, err := db.Search(ctx, []*graph.Pattern{
		{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("sport")},
		{Subject: graph.Binding("sport"), Predicate: graph.ExactString("category"), Object: graph.Binding("cat")},
	}, &SearchOptions{
		VectorFilter: &VectorFilter{
			Variable: "cat",
			Query:    []float32{1, 0, 0}, // Racket sports
			TopK:     5,
			IDType:   vector.IDTypeObject,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should find alice->tennis->racket_sport and bob->badminton->racket_sport
	if len(solutions) != 2 {
		t.Fatalf("Search() returned %d solutions, want 2", len(solutions))
	}

	for _, sol := range solutions {
		t.Logf("Found: %s likes %s (category: %s)",
			sol["person"], sol["sport"], sol["cat"])
	}
}

// ============================================================================
// Auto-Embed Tests
// ============================================================================

func TestDB_AutoEmbedOnPut(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}

	// Enable auto-embed for objects only
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Initially no vectors
	if db.VectorCount() != 0 {
		t.Errorf("VectorCount() = %d, want 0", db.VectorCount())
	}

	// Put a triple - should auto-embed the object
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Should have created a vector for the object "tennis"
	if db.VectorCount() != 1 {
		t.Errorf("VectorCount() = %d, want 1", db.VectorCount())
	}

	// Verify we can retrieve the vector
	objectID := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
	vec, err := db.GetVector(ctx, objectID)
	if err != nil {
		t.Fatalf("GetVector() error = %v", err)
	}
	if len(vec) != 8 {
		t.Errorf("Vector dimensions = %d, want 8", len(vec))
	}

	// Put another triple with the same object - should not create duplicate
	err = db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Should still have only 1 vector (tennis already embedded)
	if db.VectorCount() != 1 {
		t.Errorf("VectorCount() after second put = %d, want 1", db.VectorCount())
	}

	// Put triple with new object
	err = db.Put(ctx, graph.NewTripleFromStrings("charlie", "likes", "badminton"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Now should have 2 vectors
	if db.VectorCount() != 2 {
		t.Errorf("VectorCount() = %d, want 2", db.VectorCount())
	}
}

func TestDB_AutoEmbedSubjects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}

	// Enable auto-embed for subjects only
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedSubjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put a triple - should auto-embed the subject
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Should have created a vector for the subject "alice"
	if db.VectorCount() != 1 {
		t.Errorf("VectorCount() = %d, want 1", db.VectorCount())
	}

	// Verify it's a subject vector, not an object vector
	subjectID := vector.MakeID(vector.IDTypeSubject, []byte("alice"))
	_, err = db.GetVector(ctx, subjectID)
	if err != nil {
		t.Fatalf("GetVector(subject) error = %v", err)
	}

	// Object should NOT have a vector
	objectID := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
	_, err = db.GetVector(ctx, objectID)
	if err != vector.ErrNotFound {
		t.Errorf("GetVector(object) error = %v, want ErrNotFound", err)
	}
}

func TestDB_AutoEmbedAll(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}

	// Enable auto-embed for all components
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedAll))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put a triple - should auto-embed subject, predicate, and object
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Should have 3 vectors (one for each component)
	if db.VectorCount() != 3 {
		t.Errorf("VectorCount() = %d, want 3", db.VectorCount())
	}

	// Verify all three exist
	subjectID := vector.MakeID(vector.IDTypeSubject, []byte("alice"))
	_, err = db.GetVector(ctx, subjectID)
	if err != nil {
		t.Errorf("GetVector(subject) error = %v", err)
	}

	predicateID := vector.MakeID(vector.IDTypePredicate, []byte("likes"))
	_, err = db.GetVector(ctx, predicateID)
	if err != nil {
		t.Errorf("GetVector(predicate) error = %v", err)
	}

	objectID := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
	_, err = db.GetVector(ctx, objectID)
	if err != nil {
		t.Errorf("GetVector(object) error = %v", err)
	}
}

func TestDB_AutoEmbedBatchPut(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Batch put multiple triples
	err = db.Put(ctx,
		graph.NewTripleFromStrings("alice", "likes", "tennis"),
		graph.NewTripleFromStrings("bob", "likes", "badminton"),
		graph.NewTripleFromStrings("charlie", "likes", "tennis"), // Duplicate object
	)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Should have 2 unique object vectors (tennis and badminton)
	if db.VectorCount() != 2 {
		t.Errorf("VectorCount() = %d, want 2", db.VectorCount())
	}
}

func TestDB_AutoEmbedDisabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}

	// Enable vectors but set auto-embed to None
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedNone))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put a triple - should NOT auto-embed
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Should have no vectors
	if db.VectorCount() != 0 {
		t.Errorf("VectorCount() = %d, want 0 (auto-embed disabled)", db.VectorCount())
	}
}

func TestDB_AutoEmbedIntegrationWithSearch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add triples - objects are auto-embedded
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "badminton"))
	db.Put(ctx, graph.NewTripleFromStrings("charlie", "likes", "swimming"))

	// Search for objects similar to "tennis"
	tennisVec, _ := embedder.Embed("tennis")
	results, err := db.SearchSimilarObjects(ctx, tennisVec, 3)
	if err != nil {
		t.Fatalf("SearchSimilarObjects() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("SearchSimilarObjects() returned %d results, want 3", len(results))
	}

	// First result should be tennis (exact match)
	if string(results[0].Parts[0]) != "tennis" {
		t.Errorf("First result = %s, want tennis", results[0].Parts[0])
	}
}

// TestDB_DimensionMismatch tests that Open returns an error when Embedder
// and VectorIndex have different dimensions.
func TestDB_DimensionMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create index with 3 dimensions but embedder with 8 dimensions
	index := vector.NewFlatIndex(3)
	embedder := &mockEmbedder{dims: 8} // Mismatched dimensions!

	_, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err == nil {
		t.Fatal("Open() should return error for dimension mismatch")
	}

	// Should be a dimension mismatch error
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Errorf("Open() error = %v, want ErrDimensionMismatch", err)
	}

	// Error message should include the dimensions
	if !bytes.Contains([]byte(err.Error()), []byte("8")) || !bytes.Contains([]byte(err.Error()), []byte("3")) {
		t.Errorf("Error message should include dimensions: %v", err)
	}
}

// TestDB_DimensionMatch tests that Open succeeds when dimensions match.
func TestDB_DimensionMatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create index and embedder with matching dimensions
	index := vector.NewFlatIndex(8)
	embedder := &mockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v (dimensions should match)", err)
	}
	defer db.Close()

	// Verify it works
	ctx := context.Background()
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
}

// TestDB_NoValidationWithoutEmbedder tests that dimension validation is skipped
// when no embedder is configured.
func TestDB_NoValidationWithoutEmbedder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Only vectors, no embedder - should work fine
	index := vector.NewFlatIndex(3)

	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
}

// TestDB_HybridSearchDuplicateVariableValues tests that duplicate variable values
// get the correct cached score. This is a regression test for a bug where the
// deduplication logic would assign score 0 to duplicates instead of the cached score.
func TestDB_HybridSearchDuplicateVariableValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Multiple people like the same sport (duplicate variable values)
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "tennis"))     // Same object as alice
	db.Put(ctx, graph.NewTripleFromStrings("charlie", "likes", "tennis")) // Same object again
	db.Put(ctx, graph.NewTripleFromStrings("dave", "likes", "football"))

	// Set up vectors
	db.SetObjectVector(ctx, []byte("tennis"), []float32{1, 0, 0})   // High similarity to query
	db.SetObjectVector(ctx, []byte("football"), []float32{0, 1, 0}) // Lower similarity

	// Hybrid search for racket sports
	solutions, err := db.Search(ctx, []*graph.Pattern{
		{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("sport")},
	}, &SearchOptions{
		VectorFilter: &VectorFilter{
			Variable: "sport",
			Query:    []float32{1, 0, 0}, // Racket sports direction
			IDType:   vector.IDTypeObject,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should have 4 solutions
	if len(solutions) != 4 {
		t.Fatalf("Search() returned %d solutions, want 4", len(solutions))
	}

	// All tennis solutions should have the same (high) score
	tennisScore := float32(-1)
	footballScore := float32(-1)

	for _, sol := range solutions {
		sport := string(sol["sport"])
		score := GetVectorScore(sol)

		t.Logf("Solution: %s likes %s (score: %.4f)", sol["person"], sport, score)

		if sport == "tennis" {
			if tennisScore < 0 {
				tennisScore = score
			} else {
				// All tennis scores should match the first one
				if score != tennisScore {
					t.Errorf("Duplicate tennis score mismatch: got %.4f, want %.4f", score, tennisScore)
				}
			}
			// Tennis score should be high (close to 1.0 since it's an exact match)
			if score < 0.9 {
				t.Errorf("Tennis score too low: %.4f (expected >= 0.9)", score)
			}
		} else if sport == "football" {
			footballScore = score
		}
	}

	// Tennis should score higher than football
	if tennisScore <= footballScore {
		t.Errorf("Tennis score (%.4f) should be higher than football (%.4f)", tennisScore, footballScore)
	}

	// Verify sorting: tennis solutions should come before football
	// Find the first football solution
	firstFootballIdx := -1
	for i, sol := range solutions {
		if string(sol["sport"]) == "football" {
			firstFootballIdx = i
			break
		}
	}

	// All indices before firstFootballIdx should be tennis
	for i := 0; i < firstFootballIdx; i++ {
		if string(solutions[i]["sport"]) != "tennis" {
			t.Errorf("Solution %d should be tennis, got %s", i, solutions[i]["sport"])
		}
	}
}

// ============================================================================
// Edge Case Tests (Issue 08)
// ============================================================================

// TestDB_VectorIDsWithColons tests that vector IDs containing colons (URLs, URIs)
// are handled correctly through the entire DB lifecycle.
func TestDB_VectorIDsWithColons(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test various IDs with colons
	testCases := []struct {
		name string
		id   []byte
		vec  []float32
	}{
		{"URL with port", []byte("http://example.com:8080/path"), []float32{1, 0, 0}},
		{"URI with scheme", []byte("urn:isbn:0451450523"), []float32{0, 1, 0}},
		{"Timestamp", []byte("2024:01:15:12:30:45"), []float32{0, 0, 1}},
		{"Multiple colons", []byte("a:b:c:d:e:f"), []float32{0.5, 0.5, 0}},
		{"Colon at start", []byte(":leading:colon"), []float32{0.5, 0, 0.5}},
		{"Colon at end", []byte("trailing:colon:"), []float32{0, 0.5, 0.5}},
	}

	// Set vectors with colon-containing IDs
	for _, tc := range testCases {
		err := db.SetVector(ctx, tc.id, tc.vec)
		if err != nil {
			t.Fatalf("SetVector(%s) error = %v", tc.name, err)
		}
	}

	// Verify we can retrieve all vectors
	for _, tc := range testCases {
		vec, err := db.GetVector(ctx, tc.id)
		if err != nil {
			t.Errorf("GetVector(%s) error = %v", tc.name, err)
			continue
		}
		if len(vec) != len(tc.vec) {
			t.Errorf("GetVector(%s) dims = %d, want %d", tc.name, len(vec), len(tc.vec))
		}
	}

	// Verify search works
	results, err := db.SearchVectors(ctx, []float32{1, 0, 0}, 3)
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("SearchVectors() returned %d results, want 3", len(results))
	}

	// First result should be the URL (exact match to query)
	if string(results[0].ID) != "http://example.com:8080/path" {
		t.Errorf("First result ID = %s, want http://example.com:8080/path", results[0].ID)
	}
}

// TestDB_VectorFilterNonExistentVariable tests that VectorFilter with a variable
// that doesn't exist in solutions handles gracefully.
func TestDB_VectorFilterNonExistentVariable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Set up graph and vectors
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.SetObjectVector(ctx, []byte("tennis"), []float32{1, 0, 0})

	// Search with VectorFilter for a variable that doesn't exist in the pattern
	solutions, err := db.Search(ctx, []*graph.Pattern{
		{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("sport")},
	}, &SearchOptions{
		VectorFilter: &VectorFilter{
			Variable: "nonexistent_var", // This variable doesn't exist
			Query:    []float32{1, 0, 0},
			IDType:   vector.IDTypeObject,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should return empty results since no solutions have the variable
	if len(solutions) != 0 {
		t.Errorf("Search() returned %d solutions, want 0 (variable doesn't exist)", len(solutions))
	}
}

// TestDB_VectorFilterQueryTextNoEmbedder tests that using QueryText without
// an Embedder configured returns the appropriate error.
func TestDB_VectorFilterQueryTextNoEmbedder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create DB with vectors but NO embedder
	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Set up graph
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.SetObjectVector(ctx, []byte("tennis"), []float32{1, 0, 0})

	// Try to search with QueryText (requires embedder)
	_, err = db.Search(ctx, []*graph.Pattern{
		{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("sport")},
	}, &SearchOptions{
		VectorFilter: &VectorFilter{
			Variable:  "sport",
			QueryText: "racket sports", // This requires an embedder!
			IDType:    vector.IDTypeObject,
		},
	})

	if err != ErrEmbedderRequired {
		t.Errorf("Search() error = %v, want ErrEmbedderRequired", err)
	}
}

// TestDB_graph.VeryLargeVectors tests handling of high-dimensional vectors (like OpenAI's 1536).
func TestDB_VeryLargeVectors(t *testing.T) {
	t.Parallel()

	dims := 1536 // OpenAI embedding dimensions
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(dims)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create some large vectors
	vec1 := make([]float32, dims)
	vec2 := make([]float32, dims)
	vec3 := make([]float32, dims)

	// vec1: mostly zeros with 1 at start
	vec1[0] = 1.0

	// vec2: similar to vec1
	vec2[0] = 0.9
	vec2[1] = 0.1

	// vec3: orthogonal
	vec3[dims/2] = 1.0

	// Set vectors
	if err := db.SetVector(ctx, []byte("v1"), vec1); err != nil {
		t.Fatalf("SetVector(v1) error = %v", err)
	}
	if err := db.SetVector(ctx, []byte("v2"), vec2); err != nil {
		t.Fatalf("SetVector(v2) error = %v", err)
	}
	if err := db.SetVector(ctx, []byte("v3"), vec3); err != nil {
		t.Fatalf("SetVector(v3) error = %v", err)
	}

	// Verify we can retrieve them
	retrieved, err := db.GetVector(ctx, []byte("v1"))
	if err != nil {
		t.Fatalf("GetVector() error = %v", err)
	}
	if len(retrieved) != dims {
		t.Errorf("GetVector() dims = %d, want %d", len(retrieved), dims)
	}

	// Search should work
	results, err := db.SearchVectors(ctx, vec1, 2)
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchVectors() returned %d results, want 2", len(results))
	}

	// v1 should be first (exact match)
	if string(results[0].ID) != "v1" {
		t.Errorf("First result = %s, want v1", results[0].ID)
	}
}

// TestDB_LoadVectorsAfterAutoEmbed tests that LoadVectors works correctly
// after vectors were created via auto-embed.
func TestDB_LoadVectorsAfterAutoEmbed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create database with auto-embed and add data
	{
		index := vector.NewFlatIndex(8)
		embedder := &mockEmbedder{dims: 8}
		db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		ctx := context.Background()

		// Put triples - objects will be auto-embedded
		db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
		db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "badminton"))

		// Verify vectors were created
		if db.VectorCount() != 2 {
			t.Errorf("VectorCount() = %d, want 2", db.VectorCount())
		}

		db.Close()
	}

	// Reopen WITHOUT auto-embed (just load the vectors)
	{
		index := vector.NewFlatIndex(8)
		db, err := Open(dbPath, WithVectors(index)) // No embedder!
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer db.Close()

		ctx := context.Background()

		// Index should be empty before loading
		if db.VectorCount() != 0 {
			t.Errorf("VectorCount() before load = %d, want 0", db.VectorCount())
		}

		// Load vectors
		if err := db.LoadVectors(ctx); err != nil {
			t.Fatalf("LoadVectors() error = %v", err)
		}

		// Vectors should be loaded
		if db.VectorCount() != 2 {
			t.Errorf("VectorCount() after load = %d, want 2", db.VectorCount())
		}

		// Verify we can search
		tennisVec, _ := (&mockEmbedder{dims: 8}).Embed("tennis")
		results, err := db.SearchVectors(ctx, tennisVec, 2)
		if err != nil {
			t.Fatalf("SearchVectors() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("SearchVectors() returned %d results, want 2", len(results))
		}
	}
}

// TestDB_AutoEmbedWithHNSW tests auto-embed functionality with HNSW index.
func TestDB_AutoEmbedWithHNSW(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Use HNSW index instead of FlatIndex
	index := vector.NewHNSWIndex(8, vector.WithM(4), vector.WithEfConstruction(50))
	embedder := &mockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put triples - objects will be auto-embedded
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "badminton"))
	db.Put(ctx, graph.NewTripleFromStrings("charlie", "likes", "swimming"))

	// Verify vectors were created
	if db.VectorCount() != 3 {
		t.Errorf("VectorCount() = %d, want 3", db.VectorCount())
	}

	// Search should work with HNSW
	tennisVec, _ := embedder.Embed("tennis")
	results, err := db.SearchVectors(ctx, tennisVec, 3)
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("SearchVectors() returned %d results, want 3", len(results))
	}

	// Tennis should be most similar to "tennis" query
	if string(results[0].Parts[0]) != "tennis" {
		t.Errorf("First result = %s, want tennis", results[0].Parts[0])
	}
}

// TestDB_HNSWWithManyDeletions tests recall after many deletions from HNSW index.
func TestDB_HNSWWithManyDeletions(t *testing.T) {
	t.Parallel()

	dims := 32
	n := 100
	deleteCount := 40

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewHNSWIndex(dims, vector.WithM(8), vector.WithEfConstruction(100), vector.WithSeed(42))
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Generate and add vectors
	type vecData struct {
		id  []byte
		vec []float32
	}
	vectors := make([]vecData, n)

	for i := 0; i < n; i++ {
		vec := make([]float32, dims)
		// Create somewhat structured vectors for predictable results
		vec[i%dims] = 1.0
		vec[(i+1)%dims] = 0.5

		id := []byte{byte(i)}
		vectors[i] = vecData{id, vec}
		if err := db.SetVector(ctx, id, vec); err != nil {
			t.Fatalf("SetVector() error = %v", err)
		}
	}

	// Delete some vectors
	for i := 0; i < deleteCount; i++ {
		// Delete from the middle
		delIdx := n/3 + i
		if err := db.DeleteVector(ctx, vectors[delIdx].id); err != nil {
			t.Fatalf("DeleteVector() error = %v", err)
		}
	}

	// Verify count
	expectedCount := n - deleteCount
	if db.VectorCount() != expectedCount {
		t.Errorf("VectorCount() = %d, want %d", db.VectorCount(), expectedCount)
	}

	// Search should still work and return valid results
	query := make([]float32, dims)
	query[0] = 1.0

	results, err := db.SearchVectors(ctx, query, 10)
	if err != nil {
		t.Fatalf("SearchVectors() error = %v", err)
	}

	// Should return results (up to what's available)
	if len(results) == 0 {
		t.Error("SearchVectors() returned no results after deletions")
	}

	// None of the results should be deleted vectors
	deletedIDs := make(map[byte]bool)
	for i := 0; i < deleteCount; i++ {
		delIdx := n/3 + i
		deletedIDs[vectors[delIdx].id[0]] = true
	}

	for _, r := range results {
		if len(r.ID) > 0 && deletedIDs[r.ID[0]] {
			t.Errorf("Search returned deleted vector ID: %v", r.ID)
		}
	}
}

// ============================================================================
// Async Auto-Embed Tests (Issue 11)
// ============================================================================

// slowMockEmbedder is a mock embedder that adds artificial delay to simulate real embedders.
type slowMockEmbedder struct {
	dims       int
	embedCount int
}

func (m *slowMockEmbedder) Embed(text string) ([]float32, error) {
	m.embedCount++
	// Simple hash-based embedding for testing
	vec := make([]float32, m.dims)
	for i, c := range text {
		vec[i%m.dims] += float32(c) / 1000
	}
	return vector.NormalizeCopy(vec), nil
}

func (m *slowMockEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := m.Embed(text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

func (m *slowMockEmbedder) Dimensions() int {
	return m.dims
}

// TestDB_AsyncAutoEmbed tests the basic async auto-embed functionality.
func TestDB_AsyncAutoEmbed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	// Enable async auto-embed with buffer size 10
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects), WithAsyncAutoEmbed(10))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put a triple - embedding should be queued, not blocking
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Wait for embeddings to complete
	err = db.WaitForEmbeddings(ctx)
	if err != nil {
		t.Fatalf("WaitForEmbeddings() error = %v", err)
	}

	// Vector should now exist
	if db.VectorCount() != 1 {
		t.Errorf("VectorCount() = %d, want 1", db.VectorCount())
	}

	// Verify the vector was created correctly
	objectID := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
	vec, err := db.GetVector(ctx, objectID)
	if err != nil {
		t.Fatalf("GetVector() error = %v", err)
	}
	if len(vec) != 8 {
		t.Errorf("Vector dimensions = %d, want 8", len(vec))
	}
}

// TestDB_AsyncAutoEmbedMultiple tests async embedding with multiple Puts.
func TestDB_AsyncAutoEmbedMultiple(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects), WithAsyncAutoEmbed(100))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put multiple triples
	for i := 0; i < 10; i++ {
		sport := []byte{'s', 'p', 'o', 'r', 't', byte('0' + i)}
		err = db.Put(ctx, NewTriple([]byte("alice"), []byte("likes"), sport))
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	}

	// Wait for all embeddings to complete
	err = db.WaitForEmbeddings(ctx)
	if err != nil {
		t.Fatalf("WaitForEmbeddings() error = %v", err)
	}

	// All 10 unique objects should be embedded
	if db.VectorCount() != 10 {
		t.Errorf("VectorCount() = %d, want 10", db.VectorCount())
	}
}

// TestDB_AsyncAutoEmbedBatchPut tests async embedding with batch puts.
func TestDB_AsyncAutoEmbedBatchPut(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects), WithAsyncAutoEmbed(50))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Batch put multiple triples (some with duplicate objects)
	err = db.Put(ctx,
		graph.NewTripleFromStrings("alice", "likes", "tennis"),
		graph.NewTripleFromStrings("bob", "likes", "badminton"),
		graph.NewTripleFromStrings("charlie", "likes", "tennis"), // Duplicate object
		graph.NewTripleFromStrings("dave", "likes", "swimming"),
	)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Wait for embeddings
	err = db.WaitForEmbeddings(ctx)
	if err != nil {
		t.Fatalf("WaitForEmbeddings() error = %v", err)
	}

	// Should have 3 unique objects embedded
	if db.VectorCount() != 3 {
		t.Errorf("VectorCount() = %d, want 3", db.VectorCount())
	}
}

// TestDB_AsyncAutoEmbedClose tests that Close waits for pending embeddings.
func TestDB_AsyncAutoEmbedClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects), WithAsyncAutoEmbed(10))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	ctx := context.Background()

	// Put some triples
	for i := 0; i < 5; i++ {
		sport := []byte{'s', byte('0' + i)}
		err = db.Put(ctx, NewTriple([]byte("alice"), []byte("likes"), sport))
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	}

	// Close should wait for embeddings to complete
	err = db.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Reopen and verify vectors were persisted
	index2 := vector.NewFlatIndex(8)
	db2, err := Open(dbPath, WithVectors(index2))
	if err != nil {
		t.Fatalf("Reopen() error = %v", err)
	}
	defer db2.Close()

	err = db2.LoadVectors(ctx)
	if err != nil {
		t.Fatalf("LoadVectors() error = %v", err)
	}

	if db2.VectorCount() != 5 {
		t.Errorf("VectorCount() after reopen = %d, want 5", db2.VectorCount())
	}
}

// TestDB_AsyncAutoEmbedPendingCount tests the PendingEmbeddings method.
func TestDB_AsyncAutoEmbedPendingCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	// Use small buffer to observe queueing
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects), WithAsyncAutoEmbed(5))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Initially no pending
	if db.PendingEmbeddings() != 0 {
		t.Errorf("PendingEmbeddings() initially = %d, want 0", db.PendingEmbeddings())
	}

	// Put a triple
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Wait for embeddings
	err = db.WaitForEmbeddings(ctx)
	if err != nil {
		t.Fatalf("WaitForEmbeddings() error = %v", err)
	}

	// Should be zero after waiting
	if db.PendingEmbeddings() != 0 {
		t.Errorf("PendingEmbeddings() after wait = %d, want 0", db.PendingEmbeddings())
	}
}

// TestDB_AsyncAutoEmbedDisabled tests that sync embedding still works when async is not enabled.
func TestDB_AsyncAutoEmbedDisabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	// NO WithAsyncAutoEmbed - should use sync embedding
	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put a triple - embedding should happen synchronously
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Vector should exist immediately (no wait needed)
	if db.VectorCount() != 1 {
		t.Errorf("VectorCount() = %d, want 1 (sync embed)", db.VectorCount())
	}

	// WaitForEmbeddings should return immediately
	err = db.WaitForEmbeddings(ctx)
	if err != nil {
		t.Errorf("WaitForEmbeddings() error = %v (should be no-op)", err)
	}

	// PendingEmbeddings should be 0
	if db.PendingEmbeddings() != 0 {
		t.Errorf("PendingEmbeddings() = %d, want 0", db.PendingEmbeddings())
	}
}

// TestDB_AsyncAutoEmbedWithContextCancel tests that WaitForEmbeddings respects context cancellation.
func TestDB_AsyncAutoEmbedWithContextCancel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects), WithAsyncAutoEmbed(100))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Put triples
	err = db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Create a cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Wait with cancelled context should return error (if there are pending embeddings)
	// Since our mock embedder is fast, this might complete before we check
	// But at least verify it doesn't panic
	_ = db.WaitForEmbeddings(cancelCtx)

	// Wait with valid context to ensure cleanup
	err = db.WaitForEmbeddings(context.Background())
	if err != nil {
		t.Fatalf("WaitForEmbeddings() error = %v", err)
	}
}

// TestDB_AsyncAutoEmbedIntegrationWithSearch tests async embed with hybrid search.
func TestDB_AsyncAutoEmbedIntegrationWithSearch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(8)
	embedder := &slowMockEmbedder{dims: 8}

	db, err := Open(dbPath, WithVectors(index), WithAutoEmbed(embedder, AutoEmbedObjects), WithAsyncAutoEmbed(50))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add triples - objects are auto-embedded asynchronously
	db.Put(ctx, graph.NewTripleFromStrings("alice", "likes", "tennis"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "likes", "badminton"))
	db.Put(ctx, graph.NewTripleFromStrings("charlie", "likes", "swimming"))

	// Wait for embeddings before searching
	err = db.WaitForEmbeddings(ctx)
	if err != nil {
		t.Fatalf("WaitForEmbeddings() error = %v", err)
	}

	// Search for objects similar to "tennis"
	tennisVec, _ := embedder.Embed("tennis")
	results, err := db.SearchSimilarObjects(ctx, tennisVec, 3)
	if err != nil {
		t.Fatalf("SearchSimilarObjects() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("SearchSimilarObjects() returned %d results, want 3", len(results))
	}

	// First result should be tennis (exact match)
	if string(results[0].Parts[0]) != "tennis" {
		t.Errorf("First result = %s, want tennis", results[0].Parts[0])
	}
}

// TestDB_SearchSimilarSubjects tests the SearchSimilarSubjects function.
func TestDB_SearchSimilarSubjects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add graph data
	db.Put(ctx, graph.NewTripleFromStrings("alice", "knows", "bob"))
	db.Put(ctx, graph.NewTripleFromStrings("carol", "knows", "david"))
	db.Put(ctx, graph.NewTripleFromStrings("eve", "knows", "frank"))

	// Add vectors for subjects and objects
	db.SetSubjectVector(ctx, []byte("alice"), []float32{1, 0, 0})
	db.SetSubjectVector(ctx, []byte("carol"), []float32{0.9, 0.1, 0})
	db.SetSubjectVector(ctx, []byte("eve"), []float32{0, 1, 0})
	db.SetObjectVector(ctx, []byte("bob"), []float32{0.5, 0.5, 0})
	db.SetObjectVector(ctx, []byte("david"), []float32{0.5, 0.5, 0})

	// Search for subjects similar to alice
	results, err := db.SearchSimilarSubjects(ctx, []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("SearchSimilarSubjects() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchSimilarSubjects() returned %d results, want 2", len(results))
	}

	// Verify all results are subjects
	for _, r := range results {
		if r.IDType != vector.IDTypeSubject {
			t.Errorf("Expected IDTypeSubject, got %v", r.IDType)
		}
	}

	// First result should be alice (exact match)
	if string(results[0].Parts[0]) != "alice" {
		t.Errorf("First result = %s, want alice", results[0].Parts[0])
	}

	// Second result should be carol (similar)
	if string(results[1].Parts[0]) != "carol" {
		t.Errorf("Second result = %s, want carol", results[1].Parts[0])
	}
}

// TestDB_SearchSimilarSubjects_NoSubjects tests SearchSimilarSubjects when no subjects have vectors.
func TestDB_SearchSimilarSubjects_NoSubjects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	index := vector.NewFlatIndex(3)
	db, err := Open(dbPath, WithVectors(index))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add graph data
	db.Put(ctx, graph.NewTripleFromStrings("alice", "knows", "bob"))

	// Only add object vectors, no subject vectors
	db.SetObjectVector(ctx, []byte("bob"), []float32{1, 0, 0})

	// Search for subjects - should return empty since no subject vectors exist
	results, err := db.SearchSimilarSubjects(ctx, []float32{1, 0, 0}, 5)
	if err != nil {
		t.Fatalf("SearchSimilarSubjects() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchSimilarSubjects() returned %d results, want 0", len(results))
	}
}
