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

package vector

import (
	"bytes"
	"math"
	"math/rand"
	"sync"
	"testing"
)

// Test helper to generate random vectors
func randomVector(dims int, rng *rand.Rand) []float32 {
	v := make([]float32, dims)
	for i := range v {
		v[i] = rng.Float32()*2 - 1 // [-1, 1]
	}
	return v
}

// Test helper to generate normalized random vectors
func randomNormalizedVector(dims int, rng *rand.Rand) []float32 {
	v := randomVector(dims, rng)
	return Normalize(v)
}

// ============================================================================
// Distance function tests
// ============================================================================

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
		epsilon  float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
			epsilon:  0.0001,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
			epsilon:  0.0001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "45 degree angle",
			a:        []float32{1, 0},
			b:        []float32{1, 1},
			expected: float32(1 / math.Sqrt(2)),
			epsilon:  0.0001,
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0,
			epsilon:  0.0001,
		},
		{
			name:     "mismatched dimensions",
			a:        []float32{1, 2},
			b:        []float32{1, 2, 3},
			expected: 0,
			epsilon:  0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > float64(tt.epsilon) {
				t.Errorf("CosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCosineDistance(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	dist := Cosine(a, b)
	if dist != 0 {
		t.Errorf("Cosine() for identical vectors = %v, want 0", dist)
	}

	c := []float32{-1, 0, 0}
	dist = Cosine(a, c)
	if math.Abs(float64(dist-2)) > 0.0001 {
		t.Errorf("Cosine() for opposite vectors = %v, want 2", dist)
	}
}

func TestEuclidean(t *testing.T) {
	a := []float32{0, 0}
	b := []float32{3, 4}
	dist := Euclidean(a, b)
	// Squared euclidean distance = 3^2 + 4^2 = 25
	if dist != 25 {
		t.Errorf("Euclidean() = %v, want 25", dist)
	}
}

func TestEuclideanMismatchedDimensions(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	dist := Euclidean(a, b)
	if dist != float32(math.MaxFloat32) {
		t.Errorf("Euclidean() with mismatched dims = %v, want MaxFloat32", dist)
	}
}

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
		epsilon  float32
	}{
		{
			name:     "unit vectors same direction",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: -1.0, // Negated for distance
			epsilon:  0.0001,
		},
		{
			name:     "unit vectors opposite direction",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: 1.0, // Negated for distance
			epsilon:  0.0001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "scaled vectors",
			a:        []float32{2, 3},
			b:        []float32{4, 5},
			expected: -23.0, // -(2*4 + 3*5) = -23
			epsilon:  0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DotProduct(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > float64(tt.epsilon) {
				t.Errorf("DotProduct() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDotProductMismatchedDimensions(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	result := DotProduct(a, b)
	if result != float32(math.MaxFloat32) {
		t.Errorf("DotProduct() with mismatched dims = %v, want MaxFloat32", result)
	}
}

func TestNormalize(t *testing.T) {
	v := []float32{3, 4}
	result := NormalizeCopy(v)

	// Check norm is 1
	var norm float32
	for _, val := range result {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))
	if math.Abs(float64(norm-1)) > 0.0001 {
		t.Errorf("Normalized vector norm = %v, want 1", norm)
	}

	// Check original unchanged
	if v[0] != 3 || v[1] != 4 {
		t.Error("NormalizeCopy modified original vector")
	}
}

// ============================================================================
// Serialization tests
// ============================================================================

func TestVectorSerialization(t *testing.T) {
	original := []float32{1.5, -2.5, 3.14159, 0, -0.00001}
	bytes := VectorToBytes(original)
	restored := BytesToVector(bytes)

	if len(restored) != len(original) {
		t.Fatalf("BytesToVector length = %d, want %d", len(restored), len(original))
	}

	for i := range original {
		if restored[i] != original[i] {
			t.Errorf("BytesToVector[%d] = %v, want %v", i, restored[i], original[i])
		}
	}
}

func TestBytesToVectorInvalid(t *testing.T) {
	// Invalid length (not multiple of 4)
	result := BytesToVector([]byte{1, 2, 3})
	if result != nil {
		t.Error("BytesToVector should return nil for invalid input")
	}
}

// ============================================================================
// ID helpers tests
// ============================================================================

func TestMakeIDAndParseID(t *testing.T) {
	tests := []struct {
		name     string
		idType   IDType
		parts    [][]byte
		expected [][]byte
	}{
		{
			name:     "subject ID",
			idType:   IDTypeSubject,
			parts:    [][]byte{[]byte("alice")},
			expected: [][]byte{[]byte("alice")},
		},
		{
			name:     "triple ID",
			idType:   IDTypeTriple,
			parts:    [][]byte{[]byte("alice"), []byte("knows"), []byte("bob")},
			expected: [][]byte{[]byte("alice"), []byte("knows"), []byte("bob")},
		},
		{
			name:     "facet ID",
			idType:   IDTypeFacet,
			parts:    [][]byte{[]byte("subject"), []byte("alice"), []byte("age")},
			expected: [][]byte{[]byte("subject"), []byte("alice"), []byte("age")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := MakeID(tt.idType, tt.parts...)
			parsedType, parsedParts := ParseID(id)

			if parsedType != tt.idType {
				t.Errorf("ParseID type = %v, want %v", parsedType, tt.idType)
			}

			if len(parsedParts) != len(tt.expected) {
				t.Fatalf("ParseID parts length = %d, want %d", len(parsedParts), len(tt.expected))
			}

			for i := range tt.expected {
				if !bytes.Equal(parsedParts[i], tt.expected[i]) {
					t.Errorf("ParseID parts[%d] = %q, want %q", i, parsedParts[i], tt.expected[i])
				}
			}
		})
	}
}

func TestParseIDNoColon(t *testing.T) {
	id := []byte("rawid")
	idType, parts := ParseID(id)
	if idType != IDTypeCustom {
		t.Errorf("ParseID type = %v, want custom", idType)
	}
	if len(parts) != 1 || !bytes.Equal(parts[0], id) {
		t.Errorf("ParseID parts = %v, want %v", parts, [][]byte{id})
	}
}

// TestMakeIDParseIDWithColons tests that data containing colons is handled correctly.
// This is a regression test for the colon vulnerability where ParseID would incorrectly
// split on colons within user data (URLs, timestamps, etc.).
func TestMakeIDParseIDWithColons(t *testing.T) {
	tests := []struct {
		name   string
		idType IDType
		parts  [][]byte
	}{
		{
			name:   "URL in object",
			idType: IDTypeObject,
			parts:  [][]byte{[]byte("http://example.com:8080/path?query=value")},
		},
		{
			name:   "timestamp",
			idType: IDTypeObject,
			parts:  [][]byte{[]byte("2024:01:15:12:30:45")},
		},
		{
			name:   "multiple colons in subject",
			idType: IDTypeSubject,
			parts:  [][]byte{[]byte("foo:bar:baz:qux")},
		},
		{
			name:   "triple with URLs",
			idType: IDTypeTriple,
			parts: [][]byte{
				[]byte("http://example.com/user/1"),
				[]byte("http://schema.org/knows"),
				[]byte("http://example.com/user/2"),
			},
		},
		{
			name:   "triple with colons everywhere",
			idType: IDTypeTriple,
			parts: [][]byte{
				[]byte("user:123:abc"),
				[]byte("rel:type:subtype"),
				[]byte("target:456:def"),
			},
		},
		{
			name:   "facet with colon in name",
			idType: IDTypeFacet,
			parts:  [][]byte{[]byte("schema:org:Person")},
		},
		{
			name:   "empty part",
			idType: IDTypeObject,
			parts:  [][]byte{[]byte("")},
		},
		{
			name:   "triple with empty middle part",
			idType: IDTypeTriple,
			parts:  [][]byte{[]byte("a"), []byte(""), []byte("c")},
		},
		{
			name:   "colon only",
			idType: IDTypeObject,
			parts:  [][]byte{[]byte(":")},
		},
		{
			name:   "multiple colons only",
			idType: IDTypeObject,
			parts:  [][]byte{[]byte(":::")},
		},
		{
			name:   "binary data with colons",
			idType: IDTypeObject,
			parts:  [][]byte{{0x3A, 0x00, 0x3A, 0xFF, 0x3A}}, // 0x3A is ':'
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create ID
			id := MakeID(tt.idType, tt.parts...)

			// Parse it back
			parsedType, parsedParts := ParseID(id)

			// Verify type
			if parsedType != tt.idType {
				t.Errorf("ParseID type = %v, want %v", parsedType, tt.idType)
			}

			// Verify parts
			if len(parsedParts) != len(tt.parts) {
				t.Fatalf("ParseID parts length = %d, want %d\nID bytes: %v", len(parsedParts), len(tt.parts), id)
			}

			for i := range tt.parts {
				if !bytes.Equal(parsedParts[i], tt.parts[i]) {
					t.Errorf("ParseID parts[%d] = %q, want %q", i, parsedParts[i], tt.parts[i])
				}
			}
		})
	}
}

// TestVarintEncoding tests the internal varint helper functions.
func TestVarintEncoding(t *testing.T) {
	tests := []struct {
		value int
		size  int
	}{
		{0, 1},
		{1, 1},
		{127, 1},
		{128, 2},
		{255, 2},
		{16383, 2},
		{16384, 3},
		{1000000, 3},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// Test varintSize
			size := varintSize(tt.value)
			if size != tt.size {
				t.Errorf("varintSize(%d) = %d, want %d", tt.value, size, tt.size)
			}

			// Test roundtrip
			buf := appendVarint(nil, tt.value)
			if len(buf) != tt.size {
				t.Errorf("appendVarint length = %d, want %d", len(buf), tt.size)
			}

			value, n := readVarint(buf)
			if n != tt.size {
				t.Errorf("readVarint bytes read = %d, want %d", n, tt.size)
			}
			if value != tt.value {
				t.Errorf("readVarint value = %d, want %d", value, tt.value)
			}
		})
	}
}

// TestReadVarintErrors tests error handling in readVarint.
func TestReadVarintErrors(t *testing.T) {
	// Empty data
	val, n := readVarint([]byte{})
	if val != 0 || n != 0 {
		t.Errorf("readVarint empty = (%d, %d), want (0, 0)", val, n)
	}

	// Incomplete varint (continuation bit set but no more bytes)
	val, n = readVarint([]byte{0x80})
	if n != -1 {
		t.Errorf("readVarint incomplete = (%d, %d), want (_, -1)", val, n)
	}

	// Varint too long (overflow protection)
	longVarint := make([]byte, 11)
	for i := range longVarint {
		longVarint[i] = 0x80 // All continuation bits set
	}
	val, n = readVarint(longVarint)
	if n != -1 {
		t.Errorf("readVarint too long = (%d, %d), want (_, -1)", val, n)
	}
}

// TestParseLegacyFormat tests backward compatibility with old colon-separated IDs.
func TestParseLegacyFormat(t *testing.T) {
	// These are legacy IDs that don't use length-prefixed encoding
	// The parser should fall back to legacy parsing

	tests := []struct {
		name         string
		id           []byte
		expectedType IDType
		expectedLen  int
	}{
		{
			name:         "legacy subject (no colons in data)",
			id:           []byte("subject:alice"),
			expectedType: IDTypeSubject,
			expectedLen:  1,
		},
		{
			name:         "legacy triple",
			id:           []byte("triple:alice:knows:bob"),
			expectedType: IDTypeTriple,
			expectedLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idType, parts := ParseID(tt.id)

			if idType != tt.expectedType {
				t.Errorf("ParseID type = %v, want %v", idType, tt.expectedType)
			}

			if len(parts) != tt.expectedLen {
				t.Errorf("ParseID parts length = %d, want %d", len(parts), tt.expectedLen)
			}
		})
	}
}

// TestNormalizeScore tests the score normalization function.
func TestNormalizeScore(t *testing.T) {
	tests := []struct {
		name     string
		distance float32
		expected float32
	}{
		{"identical vectors (distance 0)", 0, 1.0},
		{"orthogonal vectors (distance 1)", 1, 0.5},
		{"opposite vectors (distance 2)", 2, 0.0},
		{"negative distance clamped", -0.5, 1.0},
		{"distance > 2 clamped", 3.0, 0.0},
		{"typical distance 0.5", 0.5, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := NormalizeScore(tt.distance)
			if math.Abs(float64(score-tt.expected)) > 0.0001 {
				t.Errorf("NormalizeScore(%v) = %v, want %v", tt.distance, score, tt.expected)
			}
		})
	}
}

// TestScoreRangeConsistency verifies that scores are consistently in [0, 1] range.
func TestScoreRangeConsistency(t *testing.T) {
	dims := 32
	rng := rand.New(rand.NewSource(42))

	// Test both FlatIndex and HNSWIndex
	indexes := []struct {
		name  string
		index Index
	}{
		{"FlatIndex", NewFlatIndex(dims)},
		{"HNSWIndex", NewHNSWIndex(dims, WithSeed(42))},
	}

	for _, idx := range indexes {
		t.Run(idx.name, func(t *testing.T) {
			// Add vectors with various relationships
			idx.index.Add([]byte("identical"), []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
			idx.index.Add([]byte("similar"), []float32{0.9, 0.1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
			idx.index.Add([]byte("orthogonal"), []float32{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
			idx.index.Add([]byte("opposite"), []float32{-1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

			// Add some random vectors
			for i := 0; i < 10; i++ {
				vec := randomNormalizedVector(dims, rng)
				idx.index.Add([]byte{byte(i)}, vec)
			}

			// Search and verify all scores are in [0, 1]
			query := []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
			results, err := idx.index.Search(query, 14)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			for _, r := range results {
				if r.Score < 0 || r.Score > 1 {
					t.Errorf("Score %v for %s is outside [0, 1] range", r.Score, r.ID)
				}
				// Also verify Distance is non-negative
				if r.Distance < 0 {
					t.Errorf("Distance %v for %s is negative", r.Distance, r.ID)
				}
			}

			// Verify scores are in descending order (higher = more similar)
			for i := 1; i < len(results); i++ {
				if results[i].Score > results[i-1].Score {
					t.Errorf("Scores not in descending order: %v > %v", results[i].Score, results[i-1].Score)
				}
			}

			// Verify identical vector has score ~1.0
			for _, r := range results {
				if string(r.ID) == "identical" {
					if r.Score < 0.99 {
						t.Errorf("Identical vector score = %v, want ~1.0", r.Score)
					}
				}
			}
		})
	}
}

// ============================================================================
// FlatIndex tests
// ============================================================================

func TestFlatIndexWithDistance(t *testing.T) {
	// Test that WithDistance option works
	idx := NewFlatIndex(3, WithDistance(Euclidean))

	// Add vectors
	idx.Add([]byte("v1"), []float32{0, 0, 0})
	idx.Add([]byte("v2"), []float32{3, 4, 0}) // Euclidean distance = 25 (squared)
	idx.Add([]byte("v3"), []float32{1, 0, 0}) // Euclidean distance = 1 (squared)

	// Search should use Euclidean distance
	results, err := idx.Search([]float32{0, 0, 0}, 3)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// v1 should be closest (distance 0), then v3 (distance 1), then v2 (distance 25)
	if len(results) != 3 {
		t.Fatalf("Search() returned %d results, want 3", len(results))
	}
	if string(results[0].ID) != "v1" {
		t.Errorf("First result = %s, want v1", results[0].ID)
	}
	if string(results[1].ID) != "v3" {
		t.Errorf("Second result = %s, want v3", results[1].ID)
	}
	if string(results[2].ID) != "v2" {
		t.Errorf("Third result = %s, want v2", results[2].ID)
	}
}

func TestFlatIndexBasicOperations(t *testing.T) {
	idx := NewFlatIndex(3)

	// Test Add
	err := idx.Add([]byte("v1"), []float32{1, 0, 0})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if idx.Len() != 1 {
		t.Errorf("Len() = %d, want 1", idx.Len())
	}

	if idx.Dimensions() != 3 {
		t.Errorf("Dimensions() = %d, want 3", idx.Dimensions())
	}

	// Test Get
	v, err := idx.Get([]byte("v1"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if v[0] != 1 || v[1] != 0 || v[2] != 0 {
		t.Errorf("Get() = %v, want [1, 0, 0]", v)
	}

	// Test Delete
	err = idx.Delete([]byte("v1"))
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if idx.Len() != 0 {
		t.Errorf("Len() after delete = %d, want 0", idx.Len())
	}

	// Test Get after delete
	_, err = idx.Get([]byte("v1"))
	if err != ErrNotFound {
		t.Errorf("Get() after delete error = %v, want ErrNotFound", err)
	}
}

func TestFlatIndexErrors(t *testing.T) {
	idx := NewFlatIndex(3)

	// Wrong dimensions
	err := idx.Add([]byte("v1"), []float32{1, 0})
	if err != ErrDimensionMismatch {
		t.Errorf("Add() wrong dims error = %v, want ErrDimensionMismatch", err)
	}

	// Empty vector
	err = idx.Add([]byte("v1"), []float32{})
	if err != ErrEmptyVector {
		t.Errorf("Add() empty error = %v, want ErrEmptyVector", err)
	}

	// Delete non-existent
	err = idx.Delete([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("Delete() non-existent error = %v, want ErrNotFound", err)
	}

	// Get non-existent
	_, err = idx.Get([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("Get() non-existent error = %v, want ErrNotFound", err)
	}

	// Search with wrong dimensions
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	_, err = idx.Search([]float32{1, 0}, 1)
	if err != ErrDimensionMismatch {
		t.Errorf("Search() wrong dims error = %v, want ErrDimensionMismatch", err)
	}

	// Search with invalid k
	_, err = idx.Search([]float32{1, 0, 0}, 0)
	if err != ErrInvalidK {
		t.Errorf("Search() k=0 error = %v, want ErrInvalidK", err)
	}
}

func TestFlatIndexSearch(t *testing.T) {
	idx := NewFlatIndex(3)

	// Add some vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0, 1, 0})
	idx.Add([]byte("v3"), []float32{0, 0, 1})
	idx.Add([]byte("v4"), []float32{0.9, 0.1, 0}) // Close to v1

	// Search for vectors similar to [1, 0, 0]
	results, err := idx.Search([]float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}

	// v1 should be first (exact match)
	if string(results[0].ID) != "v1" {
		t.Errorf("Search() first result = %s, want v1", results[0].ID)
	}

	// v4 should be second (closest after exact match)
	if string(results[1].ID) != "v4" {
		t.Errorf("Search() second result = %s, want v4", results[1].ID)
	}

	// Check score for exact match
	if results[0].Score < 0.9999 {
		t.Errorf("Search() exact match score = %v, want ~1.0", results[0].Score)
	}
}

func TestFlatIndexSearchEmpty(t *testing.T) {
	idx := NewFlatIndex(3)

	results, err := idx.Search([]float32{1, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Search() on empty index returned %d results, want 0", len(results))
	}
}

func TestFlatIndexSearchKLargerThanN(t *testing.T) {
	idx := NewFlatIndex(3)
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0, 1, 0})

	results, err := idx.Search([]float32{1, 0, 0}, 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Search() returned %d results, want 2", len(results))
	}
}

func TestFlatIndexUpdate(t *testing.T) {
	idx := NewFlatIndex(3)

	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v1"), []float32{0, 1, 0}) // Update

	v, _ := idx.Get([]byte("v1"))
	if v[0] != 0 || v[1] != 1 || v[2] != 0 {
		t.Errorf("Updated vector = %v, want [0, 1, 0]", v)
	}

	if idx.Len() != 1 {
		t.Errorf("Len() after update = %d, want 1", idx.Len())
	}
}

func TestFlatIndexConcurrency(t *testing.T) {
	idx := NewFlatIndex(32)

	var wg sync.WaitGroup
	numWriters := 10
	numReaders := 10
	opsPerWorker := 100

	// Writers - each goroutine gets its own RNG to avoid race
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(42 + workerID)))
			for i := 0; i < opsPerWorker; i++ {
				id := []byte{byte(workerID), byte(i)}
				vec := randomVector(32, rng)
				idx.Add(id, vec)
			}
		}(w)
	}

	// Readers - each goroutine gets its own RNG to avoid race
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(1000 + readerID)))
			for i := 0; i < opsPerWorker; i++ {
				query := randomVector(32, rng)
				idx.Search(query, 5)
			}
		}(r)
	}

	wg.Wait()
}

// ============================================================================
// HNSWIndex tests
// ============================================================================

func TestHNSWIndexWithEfSearch(t *testing.T) {
	// Test that WithEfSearch option works
	idx := NewHNSWIndex(3, WithSeed(42), WithM(4), WithEfConstruction(50), WithEfSearch(100))

	// Add vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0.9, 0.1, 0})
	idx.Add([]byte("v3"), []float32{0.8, 0.2, 0})

	// Search should work with the configured efSearch
	results, err := idx.Search([]float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}

	// v1 should be first (exact match)
	if string(results[0].ID) != "v1" {
		t.Errorf("First result = %s, want v1", results[0].ID)
	}
}

func TestHNSWIndexWithHNSWDistance(t *testing.T) {
	// Test that WithHNSWDistance option works with Euclidean distance
	idx := NewHNSWIndex(3, WithSeed(42), WithM(4), WithEfConstruction(50), WithHNSWDistance(Euclidean))

	// Add vectors
	idx.Add([]byte("v1"), []float32{0, 0, 0})
	idx.Add([]byte("v2"), []float32{3, 4, 0}) // Euclidean distance = 25 (squared)
	idx.Add([]byte("v3"), []float32{1, 0, 0}) // Euclidean distance = 1 (squared)

	// Search should use Euclidean distance
	results, err := idx.Search([]float32{0, 0, 0}, 3)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Search() returned %d results, want 3", len(results))
	}

	// v1 should be first (distance 0)
	if string(results[0].ID) != "v1" {
		t.Errorf("First result = %s, want v1", results[0].ID)
	}
	// v3 should be second (distance 1)
	if string(results[1].ID) != "v3" {
		t.Errorf("Second result = %s, want v3", results[1].ID)
	}
	// v2 should be third (distance 25)
	if string(results[2].ID) != "v2" {
		t.Errorf("Third result = %s, want v2", results[2].ID)
	}
}

func TestHNSWIndexBasicOperations(t *testing.T) {
	idx := NewHNSWIndex(3, WithSeed(42))

	// Test Add
	err := idx.Add([]byte("v1"), []float32{1, 0, 0})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if idx.Len() != 1 {
		t.Errorf("Len() = %d, want 1", idx.Len())
	}

	if idx.Dimensions() != 3 {
		t.Errorf("Dimensions() = %d, want 3", idx.Dimensions())
	}

	// Test Get
	v, err := idx.Get([]byte("v1"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if v[0] != 1 || v[1] != 0 || v[2] != 0 {
		t.Errorf("Get() = %v, want [1, 0, 0]", v)
	}

	// Test Delete
	err = idx.Delete([]byte("v1"))
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if idx.Len() != 0 {
		t.Errorf("Len() after delete = %d, want 0", idx.Len())
	}
}

func TestHNSWIndexSearch(t *testing.T) {
	idx := NewHNSWIndex(3, WithSeed(42), WithM(4), WithEfConstruction(50))

	// Add some vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0, 1, 0})
	idx.Add([]byte("v3"), []float32{0, 0, 1})
	idx.Add([]byte("v4"), []float32{0.9, 0.1, 0}) // Close to v1
	idx.Add([]byte("v5"), []float32{0.8, 0.2, 0}) // Also close to v1

	// Search for vectors similar to [1, 0, 0]
	results, err := idx.Search([]float32{1, 0, 0}, 3)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Search() returned %d results, want 3", len(results))
	}

	// v1 should be first (exact match)
	if string(results[0].ID) != "v1" {
		t.Errorf("Search() first result = %s, want v1", results[0].ID)
	}
}

func TestHNSWIndexLargerScale(t *testing.T) {
	dims := 32
	n := 500 // Number of vectors

	idx := NewHNSWIndex(dims, WithSeed(42), WithM(16), WithEfConstruction(100))
	flat := NewFlatIndex(dims) // For comparison

	rng := rand.New(rand.NewSource(42))

	// Add vectors to both indexes
	for i := 0; i < n; i++ {
		vec := randomNormalizedVector(dims, rng)
		id := []byte{byte(i / 256), byte(i % 256)}
		idx.Add(id, vec)
		flat.Add(id, vec)
	}

	// Generate query vectors and compare results
	numQueries := 20
	k := 10
	recallSum := 0.0

	for q := 0; q < numQueries; q++ {
		query := randomNormalizedVector(dims, rng)

		hnswResults, _ := idx.SearchWithEf(query, k, 100)
		flatResults, _ := flat.Search(query, k)

		// Calculate recall
		flatSet := make(map[string]bool)
		for _, r := range flatResults {
			flatSet[string(r.ID)] = true
		}

		hits := 0
		for _, r := range hnswResults {
			if flatSet[string(r.ID)] {
				hits++
			}
		}

		recallSum += float64(hits) / float64(k)
	}

	avgRecall := recallSum / float64(numQueries)
	t.Logf("Average recall@%d: %.2f%%", k, avgRecall*100)

	// HNSW should achieve at least 80% recall with these parameters
	if avgRecall < 0.8 {
		t.Errorf("Average recall = %.2f%%, want >= 80%%", avgRecall*100)
	}
}

func TestHNSWIndexErrors(t *testing.T) {
	idx := NewHNSWIndex(3)

	// Wrong dimensions
	err := idx.Add([]byte("v1"), []float32{1, 0})
	if err != ErrDimensionMismatch {
		t.Errorf("Add() wrong dims error = %v, want ErrDimensionMismatch", err)
	}

	// Empty vector
	err = idx.Add([]byte("v1"), []float32{})
	if err != ErrEmptyVector {
		t.Errorf("Add() empty error = %v, want ErrEmptyVector", err)
	}

	// Delete non-existent
	err = idx.Delete([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("Delete() non-existent error = %v, want ErrNotFound", err)
	}

	// Search with wrong dimensions
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	_, err = idx.Search([]float32{1, 0}, 1)
	if err != ErrDimensionMismatch {
		t.Errorf("Search() wrong dims error = %v, want ErrDimensionMismatch", err)
	}

	// Search with invalid k
	_, err = idx.Search([]float32{1, 0, 0}, 0)
	if err != ErrInvalidK {
		t.Errorf("Search() k=0 error = %v, want ErrInvalidK", err)
	}
}

func TestHNSWIndexConcurrency(t *testing.T) {
	idx := NewHNSWIndex(32, WithSeed(42))

	var wg sync.WaitGroup
	numWriters := 5
	numReaders := 10
	opsPerWorker := 50

	// Writers - each goroutine gets its own RNG to avoid race
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(42 + workerID)))
			for i := 0; i < opsPerWorker; i++ {
				id := []byte{byte(workerID), byte(i)}
				vec := randomVector(32, rng)
				idx.Add(id, vec)
			}
		}(w)
	}

	// Readers - each goroutine gets its own RNG to avoid race
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(1000 + readerID)))
			for i := 0; i < opsPerWorker; i++ {
				query := randomVector(32, rng)
				idx.Search(query, 5)
			}
		}(r)
	}

	wg.Wait()
}

func TestHNSWIndexDelete(t *testing.T) {
	idx := NewHNSWIndex(3, WithSeed(42))

	// Add vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0, 1, 0})
	idx.Add([]byte("v3"), []float32{0, 0, 1})

	// Delete v1
	err := idx.Delete([]byte("v1"))
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Search should not return v1
	results, _ := idx.Search([]float32{1, 0, 0}, 3)
	for _, r := range results {
		if string(r.ID) == "v1" {
			t.Error("Search returned deleted vector v1")
		}
	}

	if idx.Len() != 2 {
		t.Errorf("Len() = %d, want 2", idx.Len())
	}
}

// TestHNSWIndexDeleteMaintainsConnectivity tests that deleting nodes
// maintains graph connectivity by reconnecting orphaned neighbors.
func TestHNSWIndexDeleteMaintainsConnectivity(t *testing.T) {
	dims := 32
	n := 100          // Total vectors
	deleteCount := 30 // Number to delete

	idx := NewHNSWIndex(dims, WithSeed(42), WithM(8), WithEfConstruction(100))
	flat := NewFlatIndex(dims) // For comparison

	rng := rand.New(rand.NewSource(42))

	// Generate vectors
	type vecData struct {
		id  []byte
		vec []float32
	}
	vectors := make([]vecData, n)
	for i := 0; i < n; i++ {
		vec := randomNormalizedVector(dims, rng)
		id := []byte{byte(i / 256), byte(i % 256)}
		vectors[i] = vecData{id, vec}
		idx.Add(id, vec)
		flat.Add(id, vec)
	}

	// Delete some vectors from both indexes
	// Delete vectors from the middle to maximize connectivity impact
	deleteStart := n / 3
	for i := 0; i < deleteCount; i++ {
		delIdx := deleteStart + i
		idx.Delete(vectors[delIdx].id)
		flat.Delete(vectors[delIdx].id)
	}

	// Verify both indexes have the same count
	if idx.Len() != flat.Len() {
		t.Errorf("Length mismatch: HNSW=%d, Flat=%d", idx.Len(), flat.Len())
	}

	// Test recall after deletions
	numQueries := 20
	k := 10
	recallSum := 0.0

	for q := 0; q < numQueries; q++ {
		query := randomNormalizedVector(dims, rng)

		hnswResults, _ := idx.SearchWithEf(query, k, 100)
		flatResults, _ := flat.Search(query, k)

		// Calculate recall
		flatSet := make(map[string]bool)
		for _, r := range flatResults {
			flatSet[string(r.ID)] = true
		}

		hits := 0
		for _, r := range hnswResults {
			if flatSet[string(r.ID)] {
				hits++
			}
		}

		recallSum += float64(hits) / float64(min(k, len(flatResults)))
	}

	avgRecall := recallSum / float64(numQueries)
	t.Logf("Average recall@%d after %d deletions: %.2f%%", k, deleteCount, avgRecall*100)

	// After deletions, HNSW should still achieve reasonable recall (at least 70%)
	// The reconnection logic should prevent severe degradation
	if avgRecall < 0.70 {
		t.Errorf("Average recall = %.2f%%, want >= 70%% (graph connectivity may be broken)", avgRecall*100)
	}
}

// TestHNSWIndexDeleteEntryPoint tests deleting the entry point node.
func TestHNSWIndexDeleteEntryPoint(t *testing.T) {
	idx := NewHNSWIndex(3, WithSeed(42), WithM(4))

	// Add vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0, 1, 0})
	idx.Add([]byte("v3"), []float32{0, 0, 1})
	idx.Add([]byte("v4"), []float32{0.5, 0.5, 0})

	// Get the entry point ID
	idx.mu.RLock()
	entryID := idx.entryPoint.id
	idx.mu.RUnlock()

	// Delete the entry point
	err := idx.Delete([]byte(entryID))
	if err != nil {
		t.Fatalf("Delete() entry point error = %v", err)
	}

	// Index should still be searchable
	results, err := idx.Search([]float32{1, 0, 0}, 3)
	if err != nil {
		t.Fatalf("Search() after entry point deletion error = %v", err)
	}

	// Should return results from remaining vectors
	if len(results) == 0 {
		t.Error("Search() returned no results after entry point deletion")
	}

	// Deleted vector should not appear
	for _, r := range results {
		if string(r.ID) == entryID {
			t.Errorf("Search returned deleted entry point %s", entryID)
		}
	}
}

// TestHNSWIndexUpdateRebuildConnections tests that updating a vector with
// significantly different values rebuilds connections for better search quality.
func TestHNSWIndexUpdateRebuildConnections(t *testing.T) {
	dims := 32
	idx := NewHNSWIndex(dims, WithSeed(42), WithM(8), WithEfConstruction(100))
	flat := NewFlatIndex(dims) // For comparison

	rng := rand.New(rand.NewSource(42))

	// Add vectors to both indexes
	n := 50
	for i := 0; i < n; i++ {
		vec := randomNormalizedVector(dims, rng)
		id := []byte{byte(i)}
		idx.Add(id, vec)
		flat.Add(id, vec)
	}

	// Now update a vector with a significantly different value
	// Old vector was random, new vector is pointing in a specific direction
	updateID := []byte{byte(25)} // Middle vector
	newVec := make([]float32, dims)
	newVec[0] = 1.0 // Point in the first dimension direction

	idx.Add(updateID, newVec)  // Update in HNSW
	flat.Add(updateID, newVec) // Update in Flat

	// Search for the updated vector
	results, err := idx.SearchWithEf(newVec, 5, 100)
	if err != nil {
		t.Fatalf("SearchWithEf() error = %v", err)
	}

	// The updated vector should be in the top results
	foundUpdated := false
	for i, r := range results {
		if bytes.Equal(r.ID, updateID) {
			foundUpdated = true
			if i > 0 {
				t.Logf("Updated vector found at position %d (expected position 0)", i)
			}
			break
		}
	}
	if !foundUpdated {
		t.Error("Updated vector not found in search results")
	}

	// Compare recall between HNSW and FlatIndex
	flatResults, _ := flat.Search(newVec, 5)
	flatSet := make(map[string]bool)
	for _, r := range flatResults {
		flatSet[string(r.ID)] = true
	}

	hits := 0
	for _, r := range results {
		if flatSet[string(r.ID)] {
			hits++
		}
	}

	recall := float64(hits) / float64(len(flatResults))
	t.Logf("Recall after update: %.2f%%", recall*100)

	// Should achieve reasonable recall (at least 60% with these params)
	if recall < 0.6 {
		t.Errorf("Recall after update = %.2f%%, want >= 60%%", recall*100)
	}
}

// TestHNSWIndexUpdateMinorChange tests that minor vector updates
// don't trigger expensive connection rebuilds.
func TestHNSWIndexUpdateMinorChange(t *testing.T) {
	dims := 3
	idx := NewHNSWIndex(dims, WithSeed(42), WithM(4))

	// Add initial vector
	id := []byte("test")
	vec1 := []float32{1, 0, 0}
	idx.Add(id, vec1)

	// Get the original level (it shouldn't change for minor updates)
	idx.mu.RLock()
	originalLevel := idx.nodes["test"].level
	originalConnections := len(idx.nodes["test"].friends[0])
	idx.mu.RUnlock()

	// Update with a very similar vector (should NOT rebuild connections)
	vec2 := []float32{0.99, 0.01, 0} // Very similar to vec1
	idx.Add(id, vec2)

	// Check that the vector was updated
	retrieved, _ := idx.Get(id)
	if retrieved[0] != vec2[0] || retrieved[1] != vec2[1] {
		t.Error("Vector was not updated")
	}

	// Check that level is preserved (no rebuild occurred)
	idx.mu.RLock()
	newLevel := idx.nodes["test"].level
	newConnections := len(idx.nodes["test"].friends[0])
	idx.mu.RUnlock()

	if newLevel != originalLevel {
		t.Errorf("Minor update changed level from %d to %d (should be preserved)", originalLevel, newLevel)
	}
	if newConnections != originalConnections {
		t.Logf("Minor update changed connections from %d to %d", originalConnections, newConnections)
	}
}

// TestHNSWIndexUpdateMajorChange tests that major vector updates
// properly rebuild connections.
func TestHNSWIndexUpdateMajorChange(t *testing.T) {
	dims := 3
	idx := NewHNSWIndex(dims, WithSeed(42), WithM(4))

	// Add some vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0.9, 0.1, 0})
	idx.Add([]byte("v3"), []float32{0, 1, 0})
	idx.Add([]byte("v4"), []float32{0, 0, 1})

	// Update v1 to be completely different (opposite direction)
	idx.Add([]byte("v1"), []float32{-1, 0, 0})

	// Search for the new direction - v1 should be found
	results, _ := idx.Search([]float32{-1, 0, 0}, 3)

	foundV1 := false
	for _, r := range results {
		if string(r.ID) == "v1" {
			foundV1 = true
			// Should be the best match
			if r.Score < 0.99 {
				t.Errorf("v1 score = %.4f, want ~1.0 (exact match)", r.Score)
			}
			break
		}
	}

	if !foundV1 {
		t.Error("Updated v1 not found in search results for its new direction")
	}

	// Search for the OLD direction - v1 should NOT be the best match anymore
	oldDirResults, _ := idx.Search([]float32{1, 0, 0}, 2)
	if len(oldDirResults) > 0 && string(oldDirResults[0].ID) == "v1" {
		t.Error("v1 should not be the best match for its old direction after update")
	}
}

// ============================================================================
// HNSW Persistence Tests
// ============================================================================

// TestHNSWIndexExportImport tests basic export/import functionality.
func TestHNSWIndexExportImport(t *testing.T) {
	dims := 3
	idx := NewHNSWIndex(dims, WithSeed(42), WithM(4), WithEfConstruction(50))

	// Add some vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0, 1, 0})
	idx.Add([]byte("v3"), []float32{0, 0, 1})
	idx.Add([]byte("v4"), []float32{0.5, 0.5, 0})

	// Export the index
	data := idx.Export()

	// Verify export data
	if data.Dimensions != dims {
		t.Errorf("Export dimensions = %d, want %d", data.Dimensions, dims)
	}
	if len(data.Nodes) != 4 {
		t.Errorf("Export nodes = %d, want 4", len(data.Nodes))
	}
	if data.EntryPointID == "" {
		t.Error("Export entry point is empty")
	}

	// Create a new index and import
	idx2 := NewHNSWIndex(dims)
	err := idx2.Import(data)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Verify the imported index
	if idx2.Len() != 4 {
		t.Errorf("Imported index Len() = %d, want 4", idx2.Len())
	}

	// Verify we can retrieve all vectors
	for _, id := range []string{"v1", "v2", "v3", "v4"} {
		vec, err := idx2.Get([]byte(id))
		if err != nil {
			t.Errorf("Get(%s) error = %v", id, err)
		}
		if len(vec) != dims {
			t.Errorf("Get(%s) dims = %d, want %d", id, len(vec), dims)
		}
	}

	// Verify search works
	results, err := idx2.Search([]float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search() returned %d results, want 2", len(results))
	}
	if string(results[0].ID) != "v1" {
		t.Errorf("Search() first result = %s, want v1", results[0].ID)
	}
}

// TestHNSWIndexExportImportSearchQuality tests that imported index
// has the same search quality as the original.
func TestHNSWIndexExportImportSearchQuality(t *testing.T) {
	dims := 32
	n := 200

	idx := NewHNSWIndex(dims, WithSeed(42), WithM(8), WithEfConstruction(100))
	flat := NewFlatIndex(dims) // For comparison

	rng := rand.New(rand.NewSource(42))

	// Add vectors to both indexes
	for i := 0; i < n; i++ {
		vec := randomNormalizedVector(dims, rng)
		id := []byte{byte(i / 256), byte(i % 256)}
		idx.Add(id, vec)
		flat.Add(id, vec)
	}

	// Test recall before export
	k := 10
	numQueries := 10
	beforeRecall := 0.0
	queries := make([][]float32, numQueries)

	for q := 0; q < numQueries; q++ {
		queries[q] = randomNormalizedVector(dims, rng)
		hnswResults, _ := idx.SearchWithEf(queries[q], k, 100)
		flatResults, _ := flat.Search(queries[q], k)

		flatSet := make(map[string]bool)
		for _, r := range flatResults {
			flatSet[string(r.ID)] = true
		}

		hits := 0
		for _, r := range hnswResults {
			if flatSet[string(r.ID)] {
				hits++
			}
		}
		beforeRecall += float64(hits) / float64(k)
	}
	beforeRecall /= float64(numQueries)

	// Export and import
	data := idx.Export()
	idx2 := NewHNSWIndex(dims)
	if err := idx2.Import(data); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Test recall after import
	afterRecall := 0.0
	for q := 0; q < numQueries; q++ {
		hnswResults, _ := idx2.SearchWithEf(queries[q], k, 100)
		flatResults, _ := flat.Search(queries[q], k)

		flatSet := make(map[string]bool)
		for _, r := range flatResults {
			flatSet[string(r.ID)] = true
		}

		hits := 0
		for _, r := range hnswResults {
			if flatSet[string(r.ID)] {
				hits++
			}
		}
		afterRecall += float64(hits) / float64(k)
	}
	afterRecall /= float64(numQueries)

	t.Logf("Recall before export: %.2f%%, after import: %.2f%%", beforeRecall*100, afterRecall*100)

	// Recall should be identical (same graph structure)
	if math.Abs(beforeRecall-afterRecall) > 0.01 {
		t.Errorf("Recall changed significantly: before=%.2f%%, after=%.2f%%", beforeRecall*100, afterRecall*100)
	}
}

// TestHNSWIndexExportImportEmpty tests export/import of empty index.
func TestHNSWIndexExportImportEmpty(t *testing.T) {
	dims := 3
	idx := NewHNSWIndex(dims)

	// Export empty index
	data := idx.Export()
	if len(data.Nodes) != 0 {
		t.Errorf("Empty export has %d nodes, want 0", len(data.Nodes))
	}
	if data.EntryPointID != "" {
		t.Errorf("Empty export has entry point %s, want empty", data.EntryPointID)
	}

	// Import into new index
	idx2 := NewHNSWIndex(dims)
	if err := idx2.Import(data); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if idx2.Len() != 0 {
		t.Errorf("Imported empty index has %d nodes, want 0", idx2.Len())
	}
}

// TestHNSWIndexImportDimensionMismatch tests that import fails
// when dimensions don't match.
func TestHNSWIndexImportDimensionMismatch(t *testing.T) {
	idx := NewHNSWIndex(3)
	idx.Add([]byte("v1"), []float32{1, 0, 0})

	data := idx.Export()

	// Try to import into index with different dimensions
	idx2 := NewHNSWIndex(5) // Different dimensions!
	err := idx2.Import(data)
	if err != ErrDimensionMismatch {
		t.Errorf("Import() error = %v, want ErrDimensionMismatch", err)
	}
}

// TestHNSWIndexExportImportConnections tests that connections are
// properly preserved after export/import.
func TestHNSWIndexExportImportConnections(t *testing.T) {
	dims := 3
	idx := NewHNSWIndex(dims, WithSeed(42), WithM(4))

	// Add vectors
	idx.Add([]byte("v1"), []float32{1, 0, 0})
	idx.Add([]byte("v2"), []float32{0.9, 0.1, 0})
	idx.Add([]byte("v3"), []float32{0.8, 0.2, 0})
	idx.Add([]byte("v4"), []float32{0, 1, 0})

	// Get original connection count
	idx.mu.RLock()
	originalConnections := 0
	for _, node := range idx.nodes {
		for _, friends := range node.friends {
			originalConnections += len(friends)
		}
	}
	idx.mu.RUnlock()

	// Export and import
	data := idx.Export()
	idx2 := NewHNSWIndex(dims)
	if err := idx2.Import(data); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Verify connection count matches
	idx2.mu.RLock()
	importedConnections := 0
	for _, node := range idx2.nodes {
		for _, friends := range node.friends {
			importedConnections += len(friends)
		}
	}
	idx2.mu.RUnlock()

	if importedConnections != originalConnections {
		t.Errorf("Connection count: original=%d, imported=%d", originalConnections, importedConnections)
	}

	// Verify specific connections are preserved
	// v1 should be connected to v2 (they're similar)
	idx2.mu.RLock()
	v1Node := idx2.nodes["v1"]
	foundV2 := false
	if v1Node != nil {
		for _, friends := range v1Node.friends {
			if _, ok := friends["v2"]; ok {
				foundV2 = true
				break
			}
		}
	}
	idx2.mu.RUnlock()

	if !foundV2 {
		t.Error("Expected v1 to be connected to v2 after import")
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkFlatIndexAdd(b *testing.B) {
	idx := NewFlatIndex(128)
	rng := rand.New(rand.NewSource(42))
	vec := randomVector(128, rng)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := []byte{byte(i / 256), byte(i % 256)}
		idx.Add(id, vec)
	}
}

func BenchmarkFlatIndexSearch(b *testing.B) {
	idx := NewFlatIndex(128)
	rng := rand.New(rand.NewSource(42))

	// Pre-populate with 1000 vectors
	for i := 0; i < 1000; i++ {
		vec := randomVector(128, rng)
		id := []byte{byte(i / 256), byte(i % 256)}
		idx.Add(id, vec)
	}

	query := randomVector(128, rng)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search(query, 10)
	}
}

func BenchmarkHNSWIndexAdd(b *testing.B) {
	idx := NewHNSWIndex(128, WithSeed(42))
	rng := rand.New(rand.NewSource(42))
	vec := randomVector(128, rng)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := []byte{byte(i / 256), byte(i % 256)}
		idx.Add(id, vec)
	}
}

func BenchmarkHNSWIndexSearch(b *testing.B) {
	idx := NewHNSWIndex(128, WithSeed(42))
	rng := rand.New(rand.NewSource(42))

	// Pre-populate with 1000 vectors
	for i := 0; i < 1000; i++ {
		vec := randomVector(128, rng)
		id := []byte{byte(i / 256), byte(i % 256)}
		idx.Add(id, vec)
	}

	query := randomVector(128, rng)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search(query, 10)
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	a := randomVector(128, rng)
	vecB := randomVector(128, rng)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, vecB)
	}
}
