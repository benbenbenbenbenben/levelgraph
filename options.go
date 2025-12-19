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

// JoinAlgorithm represents the algorithm used for joining patterns in searches.
type JoinAlgorithm string

const (
	// JoinAlgorithmBasic uses nested loop join.
	JoinAlgorithmBasic JoinAlgorithm = "basic"
	// JoinAlgorithmSort uses sort-merge join for better performance.
	JoinAlgorithmSort JoinAlgorithm = "sort"
)

// Options configures the behavior of a LevelGraph database.
type Options struct {
	// JournalEnabled enables the journalling feature for write operations.
	JournalEnabled bool

	// FacetsEnabled enables the facets/properties feature.
	FacetsEnabled bool

	// JoinAlgorithm specifies which join algorithm to use for searches.
	// Defaults to JoinAlgorithmSort.
	JoinAlgorithm JoinAlgorithm
}

// Option is a function that configures Options.
type Option func(*Options)

// defaultOptions returns the default configuration.
func defaultOptions() *Options {
	return &Options{
		JournalEnabled: false,
		FacetsEnabled:  false,
		JoinAlgorithm:  JoinAlgorithmSort,
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
