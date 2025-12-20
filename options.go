// Copyright (c) 2013-2024 Matteo Collina and LevelGraph Contributors
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
	"log/slog"

	"github.com/benbenbenbenbenben/levelgraph/vector"
)

// JoinAlgorithm represents the algorithm used for joining patterns in searches.
type JoinAlgorithm string

const (
	// JoinAlgorithmBasic uses nested loop join.
	JoinAlgorithmBasic JoinAlgorithm = "basic"
	// JoinAlgorithmSort uses sort-merge join for better performance.
	JoinAlgorithmSort JoinAlgorithm = "sort"
)

// Options configures a LevelGraph database.
type Options struct {
	// JournalEnabled enables the journalling feature for write operations.
	JournalEnabled bool

	// FacetsEnabled enables the facets/properties feature.
	FacetsEnabled bool

	// VectorIndex is an optional vector similarity index for semantic search.
	// When set, vector operations (SetVector, GetVector, SearchVectors) are enabled.
	VectorIndex vector.Index

	// JoinAlgorithm specifies which join algorithm to use for searches.
	// Defaults to JoinAlgorithmSort.
	JoinAlgorithm JoinAlgorithm

	// Logger is an optional structured logger for debug output.
	// When nil, no logging is performed.
	Logger *slog.Logger

	// DefaultLimit is the default maximum number of results for Get/Search operations.
	// When set to a positive value, this limit is applied if no explicit limit is provided.
	// 0 means no default limit (unbounded, the default for backward compatibility).
	DefaultLimit int

	// Embedder is an optional text embedder for automatic vector generation.
	// When set along with AutoEmbedTargets, vectors are automatically created
	// when triples are added.
	Embedder Embedder

	// AutoEmbedTargets specifies which triple components should be auto-embedded.
	// Only used when Embedder is set.
	AutoEmbedTargets AutoEmbedTarget

	// AsyncAutoEmbed enables non-blocking auto-embedding.
	// When enabled, embedding is performed in a background goroutine instead of
	// blocking the Put() call. Use WaitForEmbeddings() to wait for pending work.
	AsyncAutoEmbed bool

	// AsyncEmbedBufferSize sets the buffer size for the async embed queue.
	// Defaults to 100 if not set. Only used when AsyncAutoEmbed is true.
	AsyncEmbedBufferSize int
}

// Option is a function that configures Options.
type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		JournalEnabled: false,
		FacetsEnabled:  false,
		JoinAlgorithm:  JoinAlgorithmSort,
		Logger:         nil,
	}
}

// applyOptions applies a list of option functions to an Options struct.
func applyOptions(opts ...Option) *Options {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// WithJournal enables the journalling feature.
// When enabled, all Put and Del operations are recorded in a journal
// that can be trimmed or exported.
func WithJournal() Option {
	return func(o *Options) {
		o.JournalEnabled = true
	}
}

// WithFacets enables the facets/properties feature.
// When enabled, additional properties can be attached to triple components
// or entire triples.
func WithFacets() Option {
	return func(o *Options) {
		o.FacetsEnabled = true
	}
}

// WithJoinAlgorithm sets the join algorithm for searches.
func WithJoinAlgorithm(algo JoinAlgorithm) Option {
	return func(o *Options) {
		o.JoinAlgorithm = algo
	}
}

// WithBasicJoin is a convenience option for using the basic (nested loop) join algorithm.
func WithBasicJoin() Option {
	return WithJoinAlgorithm(JoinAlgorithmBasic)
}

// WithSortJoin is a convenience option for using the sort-merge join algorithm.
func WithSortJoin() Option {
	return WithJoinAlgorithm(JoinAlgorithmSort)
}

// WithLogger sets an optional structured logger for debug output.
// Pass nil to disable logging (the default).
func WithLogger(l *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// WithDefaultLimit sets the default maximum result limit for Get/Search operations.
// When set to a positive value, this limit is applied if no explicit limit is provided
// in the query. This is useful for preventing unbounded result sets that could
// exhaust memory or cause performance issues.
// 0 means no default limit (the default for backward compatibility).
func WithDefaultLimit(limit int) Option {
	return func(o *Options) {
		o.DefaultLimit = limit
	}
}

// WithVectors enables vector similarity search with the provided index.
// Use vector.NewFlatIndex for exact search or vector.NewHNSWIndex for
// approximate nearest neighbor search.
//
// Example:
//
//	db, err := levelgraph.Open("/path/to/db",
//	    levelgraph.WithVectors(vector.NewHNSWIndex(192)),
//	)
func WithVectors(index vector.Index) Option {
	return func(o *Options) {
		o.VectorIndex = index
	}
}

// Embedder is an interface for text embedding models.
// Implementations convert text to vector representations for semantic search.
type Embedder interface {
	// Embed converts a single text string to a vector embedding.
	Embed(text string) ([]float32, error)

	// EmbedBatch converts multiple texts to vector embeddings.
	// Implementations may optimize batch processing.
	EmbedBatch(texts []string) ([][]float32, error)

	// Dimensions returns the dimensionality of the embeddings.
	Dimensions() int
}

// AutoEmbedTarget specifies which parts of triples should be automatically embedded.
type AutoEmbedTarget int

const (
	// AutoEmbedNone disables automatic embedding.
	AutoEmbedNone AutoEmbedTarget = 0
	// AutoEmbedSubjects enables automatic embedding of subject values.
	AutoEmbedSubjects AutoEmbedTarget = 1 << iota
	// AutoEmbedPredicates enables automatic embedding of predicate values.
	AutoEmbedPredicates
	// AutoEmbedObjects enables automatic embedding of object values.
	AutoEmbedObjects
	// AutoEmbedAll enables automatic embedding of all triple components.
	AutoEmbedAll = AutoEmbedSubjects | AutoEmbedPredicates | AutoEmbedObjects
)

// WithAutoEmbed enables automatic vector embedding when triples are added.
// Requires both an Embedder and a VectorIndex to be configured.
//
// Example:
//
//	db, err := levelgraph.Open("/path/to/db",
//	    levelgraph.WithVectors(vector.NewHNSWIndex(192)),
//	    levelgraph.WithAutoEmbed(myEmbedder, levelgraph.AutoEmbedObjects),
//	)
func WithAutoEmbed(embedder Embedder, targets AutoEmbedTarget) Option {
	return func(o *Options) {
		o.Embedder = embedder
		o.AutoEmbedTargets = targets
	}
}

// WithAsyncAutoEmbed enables non-blocking auto-embedding with the specified buffer size.
// When enabled, embedding is performed in a background goroutine instead of blocking
// the Put() call. This is useful when using real embedding models that have latency.
//
// Use WaitForEmbeddings() to block until all pending embeddings are complete.
// The buffer size determines how many embedding requests can be queued before Put()
// blocks waiting for the queue to drain.
//
// Example:
//
//	db, err := levelgraph.Open("/path/to/db",
//	    levelgraph.WithVectors(vector.NewHNSWIndex(192)),
//	    levelgraph.WithAutoEmbed(myEmbedder, levelgraph.AutoEmbedObjects),
//	    levelgraph.WithAsyncAutoEmbed(100),
//	)
//	// ... add triples ...
//	db.WaitForEmbeddings(ctx) // Wait for all embeddings to complete
func WithAsyncAutoEmbed(bufferSize int) Option {
	return func(o *Options) {
		o.AsyncAutoEmbed = true
		o.AsyncEmbedBufferSize = bufferSize
	}
}
