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

// Package vector provides vector similarity search capabilities for LevelGraph.
//
// This package enables semantic search by associating vector embeddings with
// graph elements (subjects, objects, predicates, triples, or facets).
//
// # Index Types
//
// Two index implementations are provided:
//
//   - FlatIndex: Brute-force exact nearest neighbor search. Best for small datasets
//     (< 10,000 vectors) or when 100% recall is required. O(n) search time.
//
//   - HNSWIndex: Hierarchical Navigable Small World graphs for approximate nearest
//     neighbor search. Best for larger datasets. O(log n) search time with high recall.
//
// # Score Ranges
//
// All search results include both Distance and Score fields:
//
//   - Distance: Raw cosine distance in range [0, 2]. 0 means identical vectors,
//     2 means opposite vectors.
//
//   - Score: Normalized similarity in range [0, 1]. 1 means identical vectors,
//     0 means maximally dissimilar. Computed as: score = 1 - distance/2
//
// Use Score for filtering (e.g., MinScore: 0.8) and Distance for debugging.
//
// # Basic Usage
//
//	// Create a vector index
//	index := vector.NewFlatIndex(192) // 192 dimensions
//
//	// Add vectors
//	index.Add([]byte("doc:1"), embedding1)
//	index.Add([]byte("doc:2"), embedding2)
//
//	// Search for similar vectors
//	results, err := index.Search(queryVector, 10)
//	for _, match := range results {
//	    fmt.Printf("ID: %s, Score: %.3f\n", match.ID, match.Score)
//	}
//
// # HNSW Index
//
// For high-performance approximate nearest neighbor search, use HNSW:
//
//	index := vector.NewHNSWIndex(192,
//	    vector.WithM(16),             // Connections per node (12-48)
//	    vector.WithEfConstruction(200), // Build quality (>= 2*M)
//	    vector.WithEfSearch(100),      // Search quality (higher = better recall)
//	)
//
// See NewHNSWIndex for detailed parameter tuning guidance.
//
// # Integration with LevelGraph
//
// Use WithVectors to enable vector search in LevelGraph:
//
//	db, err := levelgraph.Open("/path/to/db",
//	    levelgraph.WithVectors(vector.NewHNSWIndex(192)),
//	    levelgraph.WithAutoEmbed(embedder, levelgraph.AutoEmbedObjects),
//	)
//
//	// Hybrid search: graph + vector similarity
//	solutions, err := db.Search(ctx, []*levelgraph.Pattern{
//	    {Subject: levelgraph.V("person"), Predicate: []byte("likes"), Object: levelgraph.V("topic")},
//	}, &levelgraph.SearchOptions{
//	    VectorFilter: &levelgraph.VectorFilter{
//	        Variable: "topic",
//	        QueryText: "machine learning",
//	        TopK: 10,
//	        MinScore: 0.7,
//	    },
//	})
package vector

import (
	"encoding/binary"
	"errors"
	"math"
)

var (
	// ErrDimensionMismatch is returned when vector dimensions don't match.
	ErrDimensionMismatch = errors.New("vector: dimension mismatch")
	// ErrNotFound is returned when a vector ID is not found.
	ErrNotFound = errors.New("vector: not found")
	// ErrEmptyVector is returned when an empty vector is provided.
	ErrEmptyVector = errors.New("vector: empty vector")
	// ErrInvalidK is returned when k <= 0 in search.
	ErrInvalidK = errors.New("vector: k must be positive")
)

// Index is the interface for vector similarity indexes.
// Implementations must be safe for concurrent use.
type Index interface {
	// Add adds or updates a vector with the given ID.
	// Returns ErrDimensionMismatch if the vector has wrong dimensions.
	Add(id []byte, vector []float32) error

	// Delete removes a vector by ID.
	// Returns ErrNotFound if the ID doesn't exist.
	Delete(id []byte) error

	// Search finds the k nearest vectors to the query.
	// Returns results sorted by distance (ascending).
	Search(query []float32, k int) ([]Match, error)

	// Get retrieves a vector by ID.
	// Returns ErrNotFound if the ID doesn't exist.
	Get(id []byte) ([]float32, error)

	// Len returns the number of vectors in the index.
	Len() int

	// Dimensions returns the vector dimensionality.
	Dimensions() int
}

// Match represents a search result with ID and similarity score.
type Match struct {
	// ID is the identifier of the matched vector.
	ID []byte
	// Score is the similarity score normalized to [0, 1] range.
	// 0 means completely dissimilar, 1 means identical.
	// For cosine distance, this is (1 - distance) / 2, mapping cosine
	// similarity from [-1, 1] to [0, 1].
	Score float32
	// Distance is the raw distance metric (lower is more similar).
	// For cosine distance, this is in range [0, 2].
	Distance float32
}

// NormalizeScore converts a cosine distance to a normalized [0, 1] score.
// Cosine distance is in range [0, 2], so score = (2 - distance) / 2 = 1 - distance/2.
func NormalizeScore(distance float32) float32 {
	// Clamp distance to valid range [0, 2]
	if distance < 0 {
		distance = 0
	}
	if distance > 2 {
		distance = 2
	}
	return 1 - distance/2
}

// DistanceFunc computes the distance between two vectors.
// Lower values indicate more similar vectors.
type DistanceFunc func(a, b []float32) float32

// Cosine computes the cosine distance (1 - cosine_similarity).
// Returns 0 for identical vectors, 2 for opposite vectors.
func Cosine(a, b []float32) float32 {
	return 1 - CosineSimilarity(a, b)
}

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value in range [-1, 1], where 1 means identical direction.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / float32(math.Sqrt(float64(normA)*float64(normB)))
}

// Euclidean computes the squared Euclidean distance.
// Using squared distance avoids the sqrt for performance.
func Euclidean(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.MaxFloat32)
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum
}

// DotProduct computes the negative dot product (for use as distance).
// Larger dot products become smaller distances.
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.MaxFloat32)
	}

	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return -sum // Negate so that larger dot products = smaller distance
}

// Normalize normalizes a vector to unit length (L2 norm).
// Modifies the vector in place and returns it.
func Normalize(v []float32) []float32 {
	var norm float32
	for _, val := range v {
		norm += val * val
	}
	if norm == 0 {
		return v
	}
	norm = float32(math.Sqrt(float64(norm)))
	for i := range v {
		v[i] /= norm
	}
	return v
}

// NormalizeCopy returns a normalized copy of the vector.
func NormalizeCopy(v []float32) []float32 {
	result := make([]float32, len(v))
	copy(result, v)
	return Normalize(result)
}

// VectorToBytes serializes a float32 vector to bytes.
func VectorToBytes(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, val := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(val))
	}
	return buf
}

// BytesToVector deserializes bytes to a float32 vector.
func BytesToVector(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}

// IDType represents what kind of graph element a vector ID refers to.
type IDType string

const (
	// IDTypeSubject identifies a subject value.
	IDTypeSubject IDType = "subject"
	// IDTypePredicate identifies a predicate value.
	IDTypePredicate IDType = "predicate"
	// IDTypeObject identifies an object value.
	IDTypeObject IDType = "object"
	// IDTypeTriple identifies a complete triple.
	IDTypeTriple IDType = "triple"
	// IDTypeFacet identifies a facet value.
	IDTypeFacet IDType = "facet"
	// IDTypeCustom identifies a custom/user-defined ID.
	IDTypeCustom IDType = "custom"
)

// MakeID creates a typed vector ID for graph elements.
// Format: "type:" followed by length-prefixed parts.
// Each part is encoded as: varint(length) + data bytes.
// This allows any byte sequence (including colons) in the data.
func MakeID(idType IDType, parts ...[]byte) []byte {
	// Calculate size: type + ":" + length-prefixed parts
	size := len(idType) + 1 // type + ":"
	for _, p := range parts {
		size += varintSize(len(p)) + len(p)
	}

	result := make([]byte, 0, size)
	result = append(result, idType...)
	result = append(result, ':')
	for _, p := range parts {
		result = appendVarint(result, len(p))
		result = append(result, p...)
	}
	return result
}

// ParseID extracts the type and value parts from a vector ID.
// Handles both new length-prefixed format and legacy colon-separated format.
func ParseID(id []byte) (IDType, [][]byte) {
	// Find first colon (separates type from parts)
	colonIdx := -1
	for i, b := range id {
		if b == ':' {
			colonIdx = i
			break
		}
	}

	if colonIdx == -1 {
		return IDTypeCustom, [][]byte{id}
	}

	idType := IDType(id[:colonIdx])
	rest := id[colonIdx+1:]

	if len(rest) == 0 {
		return idType, nil
	}

	// Try to parse as length-prefixed format
	parts, ok := parseLengthPrefixed(rest)
	if ok {
		return idType, parts
	}

	// Fallback: legacy colon-separated format (for backward compatibility)
	// Only used if length-prefixed parsing fails
	return parseLegacyFormat(idType, rest)
}

// parseLengthPrefixed attempts to parse data as length-prefixed parts.
// Returns (parts, true) on success, (nil, false) if format doesn't match.
func parseLengthPrefixed(data []byte) ([][]byte, bool) {
	var parts [][]byte
	offset := 0

	for offset < len(data) {
		length, n := readVarint(data[offset:])
		if n <= 0 {
			return nil, false // Invalid varint
		}
		offset += n

		if offset+length > len(data) {
			return nil, false // Length exceeds remaining data
		}

		part := make([]byte, length)
		copy(part, data[offset:offset+length])
		parts = append(parts, part)
		offset += length
	}

	return parts, true
}

// parseLegacyFormat handles old colon-separated IDs for backward compatibility.
// This is only used when length-prefixed parsing fails.
func parseLegacyFormat(idType IDType, rest []byte) (IDType, [][]byte) {
	// For single-part types, the entire rest is one part (preserves colons in data)
	switch idType {
	case IDTypeSubject, IDTypePredicate, IDTypeObject, IDTypeFacet, IDTypeCustom:
		return idType, [][]byte{rest}
	}

	// For IDTypeTriple, we need exactly 3 parts - split on first two colons only
	if idType == IDTypeTriple {
		parts := splitN(rest, ':', 3)
		return idType, parts
	}

	// Unknown type - return as single part to be safe
	return idType, [][]byte{rest}
}

// splitN splits data on separator, returning at most n parts.
// The last part contains the remainder (may include separators).
func splitN(data []byte, sep byte, n int) [][]byte {
	if n <= 0 {
		return nil
	}

	var parts [][]byte
	start := 0
	for i := 0; i < len(data) && len(parts) < n-1; i++ {
		if data[i] == sep {
			parts = append(parts, data[start:i])
			start = i + 1
		}
	}
	parts = append(parts, data[start:])
	return parts
}

// varintSize returns the number of bytes needed to encode n as a varint.
func varintSize(n int) int {
	size := 1
	for n >= 0x80 {
		n >>= 7
		size++
	}
	return size
}

// appendVarint appends n as a varint to buf and returns the extended buffer.
func appendVarint(buf []byte, n int) []byte {
	for n >= 0x80 {
		buf = append(buf, byte(n)|0x80)
		n >>= 7
	}
	buf = append(buf, byte(n))
	return buf
}

// readVarint reads a varint from data, returning (value, bytes_read).
// Returns (0, 0) if data is empty, (0, -1) if varint is malformed.
func readVarint(data []byte) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}

	var result int
	var shift uint
	for i, b := range data {
		if i >= 10 { // Varint too long (overflow protection)
			return 0, -1
		}
		result |= int(b&0x7F) << shift
		if b < 0x80 {
			return result, i + 1
		}
		shift += 7
	}
	return 0, -1 // Incomplete varint
}
