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
	"container/heap"
	"sync"
)

// FlatIndex is a brute-force vector index that computes exact nearest neighbors.
// It is WASM-compatible and suitable for small to medium datasets (< 100k vectors).
// For larger datasets, consider using HNSWIndex for approximate search.
type FlatIndex struct {
	dimensions int
	distance   DistanceFunc
	vectors    map[string][]float32
	mu         sync.RWMutex
}

// FlatOption configures a FlatIndex.
type FlatOption func(*FlatIndex)

// WithDistance sets the distance function for the flat index.
// Default is Cosine distance.
func WithDistance(fn DistanceFunc) FlatOption {
	return func(f *FlatIndex) {
		f.distance = fn
	}
}

// NewFlatIndex creates a new brute-force vector index.
// This provides exact nearest neighbor search with O(n) query time.
func NewFlatIndex(dimensions int, opts ...FlatOption) *FlatIndex {
	f := &FlatIndex{
		dimensions: dimensions,
		distance:   Cosine,
		vectors:    make(map[string][]float32),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Add adds or updates a vector with the given ID.
func (f *FlatIndex) Add(id []byte, vector []float32) error {
	if len(vector) == 0 {
		return ErrEmptyVector
	}
	if len(vector) != f.dimensions {
		return ErrDimensionMismatch
	}

	// Make a copy to avoid external modification
	v := make([]float32, len(vector))
	copy(v, vector)

	f.mu.Lock()
	f.vectors[string(id)] = v
	f.mu.Unlock()

	return nil
}

// Delete removes a vector by ID.
func (f *FlatIndex) Delete(id []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := string(id)
	if _, exists := f.vectors[key]; !exists {
		return ErrNotFound
	}
	delete(f.vectors, key)
	return nil
}

// Get retrieves a vector by ID.
func (f *FlatIndex) Get(id []byte) ([]float32, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, exists := f.vectors[string(id)]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy
	result := make([]float32, len(v))
	copy(result, v)
	return result, nil
}

// Search finds the k nearest vectors to the query.
func (f *FlatIndex) Search(query []float32, k int) ([]Match, error) {
	if k <= 0 {
		return nil, ErrInvalidK
	}
	if len(query) != f.dimensions {
		return nil, ErrDimensionMismatch
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if len(f.vectors) == 0 {
		return []Match{}, nil
	}

	// Use a max-heap to keep track of the k smallest distances
	h := &matchHeap{}
	heap.Init(h)

	for idStr, vec := range f.vectors {
		dist := f.distance(query, vec)

		if h.Len() < k {
			heap.Push(h, matchEntry{
				id:       idStr,
				distance: dist,
			})
		} else if dist < (*h)[0].distance {
			heap.Pop(h)
			heap.Push(h, matchEntry{
				id:       idStr,
				distance: dist,
			})
		}
	}

	// Extract results in sorted order (ascending by distance)
	results := make([]Match, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		entry := heap.Pop(h).(matchEntry)
		results[i] = Match{
			ID:       []byte(entry.id),
			Distance: entry.distance,
			Score:    NormalizeScore(entry.distance),
		}
	}

	return results, nil
}

// Len returns the number of vectors in the index.
func (f *FlatIndex) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.vectors)
}

// Dimensions returns the vector dimensionality.
func (f *FlatIndex) Dimensions() int {
	return f.dimensions
}

// matchEntry is an internal type for the heap.
type matchEntry struct {
	id       string
	distance float32
}

// matchHeap is a max-heap of match entries (by distance).
// We use a max-heap so we can efficiently remove the farthest entry
// when we find a closer one.
type matchHeap []matchEntry

func (h matchHeap) Len() int           { return len(h) }
func (h matchHeap) Less(i, j int) bool { return h[i].distance > h[j].distance } // Max-heap
func (h matchHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *matchHeap) Push(x any) {
	*h = append(*h, x.(matchEntry))
}

func (h *matchHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Ensure FlatIndex implements Index.
var _ Index = (*FlatIndex)(nil)
