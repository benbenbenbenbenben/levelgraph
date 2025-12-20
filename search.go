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
	"context"
	"sort"

	"github.com/benbenbenbenbenben/levelgraph/vector"
)

// VectorFilter specifies how to filter search results using vector similarity.
// It allows hybrid queries that combine graph traversal with semantic search.
//
// # How Hybrid Search Works
//
// Hybrid search executes in two phases:
//  1. Graph phase: Executes patterns to find matching solutions (like regular Search)
//  2. Vector phase: Scores and filters solutions based on vector similarity
//
// The Variable field specifies which solution variable to look up in the vector index.
// For example, if your pattern binds ?topic, and you have vectors stored for topics,
// set Variable: "topic" to score solutions by topic similarity.
//
// # Score Interpretation
//
// Scores are normalized to [0, 1] range:
//   - 1.0: Identical vectors (perfect match)
//   - 0.7-0.9: Highly similar (typically good matches)
//   - 0.5-0.7: Moderately similar
//   - 0.0-0.5: Dissimilar
//
// Use MinScore to filter out low-quality matches.
//
// # Example: Find People Who Like Similar Topics
//
//	solutions, err := db.Search(ctx, []*Pattern{
//	    {Subject: V("person"), Predicate: []byte("likes"), Object: V("topic")},
//	}, &SearchOptions{
//	    VectorFilter: &VectorFilter{
//	        Variable:  "topic",
//	        QueryText: "machine learning",  // Requires configured Embedder
//	        TopK:      10,                   // Limit to top 10 similar topics
//	        MinScore:  0.7,                  // Filter out scores below 0.7
//	        IDType:    vector.IDTypeObject, // Look up topic as object vector
//	    },
//	})
//
// # Example: Vector Search with Precomputed Query
//
//	queryVec := embedder.Embed("artificial intelligence")
//	solutions, err := db.Search(ctx, patterns, &SearchOptions{
//	    VectorFilter: &VectorFilter{
//	        Variable: "topic",
//	        Query:    queryVec,  // Use precomputed vector
//	        TopK:     10,
//	    },
//	})
type VectorFilter struct {
	// Variable is the name of the variable to filter by vector similarity.
	// The variable's value will be used to look up vectors in the index.
	Variable string

	// Query is the query vector to compare against.
	Query []float32

	// QueryText is an optional text query that will be embedded using the
	// configured Embedder. Either Query or QueryText should be set, not both.
	QueryText string

	// TopK limits results to the K most similar values for the variable.
	// If 0, all solutions are kept but scored/sorted.
	TopK int

	// MinScore filters out solutions where the similarity score is below this threshold.
	// Score is in range [0, 1] for cosine similarity (after normalization).
	MinScore float32

	// IDType specifies the type of vector ID to look up (e.g., IDTypeObject).
	// If empty, defaults to IDTypeObject.
	IDType vector.IDType
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	// Limit restricts the number of results (0 means no limit)
	Limit int
	// Offset skips the first N results
	Offset int
	// Filter is an optional function to filter solutions
	Filter func(Solution) bool
	// AsyncFilter is an optional async filter (returns solution or nil)
	AsyncFilter func(Solution, func(Solution, error))
	// Materialized is a pattern to transform solutions into triples
	Materialized *Pattern
	// InitialSolution is an optional starting solution with pre-bound variables
	InitialSolution Solution
	// VectorFilter enables hybrid search by filtering/ranking solutions based
	// on vector similarity of a bound variable.
	VectorFilter *VectorFilter
}

// Search executes a search query with one or more patterns.
// It performs joins across patterns, binding variables as it matches triples.
func (db *DB) Search(ctx context.Context, patterns []*Pattern, opts *SearchOptions) ([]Solution, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	if len(patterns) == 0 {
		return []Solution{}, nil
	}

	if opts == nil {
		opts = &SearchOptions{}
	}

	// Start with initial solution or empty solution
	var startSolution Solution
	if opts.InitialSolution != nil {
		startSolution = opts.InitialSolution.Clone()
	} else {
		startSolution = make(Solution)
	}
	solutions := []Solution{startSolution}

	// Process each pattern in sequence, joining with previous solutions
	for _, pattern := range patterns {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var newSolutions []Solution

		for _, solution := range solutions {
			// Update the pattern with bound variables from the current solution
			updatedPattern := pattern.UpdateWithSolution(solution)

			// Get matching triples (use internal method that doesn't re-lock)
			triples, err := db.getUnlocked(updatedPattern)
			if err != nil {
				return nil, err
			}

			// Bind each matching triple to the solution
			for _, triple := range triples {
				newSolution := pattern.BindTriple(solution, triple)
				if newSolution != nil {
					// Apply pattern-level filter if present
					if pattern.Filter == nil || pattern.Filter(triple) {
						newSolutions = append(newSolutions, newSolution)
					}
				}
			}
		}

		solutions = newSolutions
		if len(solutions) == 0 {
			break
		}
	}

	// Apply solution-level filter
	if opts.Filter != nil {
		var filtered []Solution
		for _, s := range solutions {
			if opts.Filter(s) {
				filtered = append(filtered, s)
			}
		}
		solutions = filtered
	}

	// Apply vector filter for hybrid search
	if opts.VectorFilter != nil && db.options.VectorIndex != nil {
		var err error
		solutions, err = db.applyVectorFilter(ctx, solutions, opts.VectorFilter)
		if err != nil {
			return nil, err
		}
	}

	// Apply offset
	if opts.Offset > 0 {
		if opts.Offset >= len(solutions) {
			solutions = []Solution{}
		} else {
			solutions = solutions[opts.Offset:]
		}
	}

	// Apply limit (use default limit if no explicit limit provided)
	limit := opts.Limit
	if limit <= 0 && db.options.DefaultLimit > 0 {
		limit = db.options.DefaultLimit
	}
	if limit > 0 && limit < len(solutions) {
		solutions = solutions[:limit]
	}

	// Apply materialization if requested
	if opts.Materialized != nil {
		return db.materializeSolutions(solutions, opts.Materialized)
	}

	return solutions, nil
}

// materializeSolutions transforms solutions into triples based on a pattern.
func (db *DB) materializeSolutions(solutions []Solution, pattern *Pattern) ([]Solution, error) {
	var result []Solution

	for _, solution := range solutions {
		// Create a new solution that represents a materialized triple
		tripleData := make(Solution)

		// Subject
		if v := pattern.GetVariable("subject"); v != nil {
			if val, ok := solution[v.Name]; ok {
				tripleData["subject"] = val
			}
		} else if val := pattern.GetConcreteValue("subject"); val != nil {
			tripleData["subject"] = val
		}

		// Predicate
		if v := pattern.GetVariable("predicate"); v != nil {
			if val, ok := solution[v.Name]; ok {
				tripleData["predicate"] = val
			}
		} else if val := pattern.GetConcreteValue("predicate"); val != nil {
			tripleData["predicate"] = val
		}

		// Object
		if v := pattern.GetVariable("object"); v != nil {
			if val, ok := solution[v.Name]; ok {
				tripleData["object"] = val
			}
		} else if val := pattern.GetConcreteValue("object"); val != nil {
			tripleData["object"] = val
		}

		result = append(result, tripleData)
	}

	return result, nil
}

// SearchIterator returns an iterator for search results.
func (db *DB) SearchIterator(ctx context.Context, patterns []*Pattern, opts *SearchOptions) (*SolutionIterator, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}

	var startSolution Solution
	if opts.InitialSolution != nil {
		startSolution = opts.InitialSolution.Clone()
	} else {
		startSolution = make(Solution)
	}

	si := &SolutionIterator{
		ctx:       ctx,
		db:        db,
		patterns:  patterns,
		opts:      opts,
		iters:     make([]*TripleIterator, len(patterns)),
		solutions: make([]Solution, len(patterns)+1),
	}
	si.solutions[0] = startSolution

	return si, nil
}

// SolutionIterator iterates over search solutions.
type SolutionIterator struct {
	ctx       context.Context
	db        *DB
	patterns  []*Pattern
	opts      *SearchOptions
	iters     []*TripleIterator
	solutions []Solution // solutions[i] is the solution before pattern[i]
	current   Solution
	err       error
	count     int
	skipped   int
	closed    bool
}

// Next advances to the next solution.
func (si *SolutionIterator) Next() bool {
	if si.closed || si.err != nil {
		return false
	}

	if si.opts.Limit > 0 && si.count >= si.opts.Limit {
		return false
	}

	for {
		select {
		case <-si.ctx.Done():
			si.err = si.ctx.Err()
			return false
		default:
		}

		solution := si.advance()
		if solution == nil {
			si.Close()
			return false
		}

		// Apply solution-level filter
		if si.opts.Filter != nil && !si.opts.Filter(solution) {
			continue
		}

		// Handle offset
		if si.skipped < si.opts.Offset {
			si.skipped++
			continue
		}

		// Apply materialization if requested
		if si.opts.Materialized != nil {
			materialized := si.materialize(solution, si.opts.Materialized)
			si.current = materialized
		} else {
			si.current = solution
		}

		si.count++
		return true
	}
}

func (si *SolutionIterator) advance() Solution {
	level := -1
	// Find the deepest active level
	for i := len(si.patterns) - 1; i >= 0; i-- {
		if si.iters[i] != nil {
			level = i
			break
		}
	}

	// If no levels are active, start at level 0
	if level == -1 {
		if len(si.patterns) == 0 {
			// Special case: no patterns, return the initial solution once
			if si.solutions[0] != nil {
				sol := si.solutions[0]
				si.solutions[0] = nil
				return sol
			}
			return nil
		}
		level = 0
		updatedPattern := si.patterns[0].UpdateWithSolution(si.solutions[0])
		iter, err := si.db.GetIterator(si.ctx, updatedPattern)
		if err != nil {
			si.err = err
			return nil
		}
		si.iters[0] = iter
	}

	for level >= 0 {
		if si.iters[level].Next() {
			triple, err := si.iters[level].Triple()
			if err != nil {
				si.err = err
				return nil
			}

			newSolution := si.patterns[level].BindTriple(si.solutions[level], triple)
			if newSolution == nil {
				continue
			}

			// Apply pattern-level filter if present
			if si.patterns[level].Filter != nil && !si.patterns[level].Filter(triple) {
				continue
			}

			if level == len(si.patterns)-1 {
				// We found a full solution!
				return newSolution
			}

			// Move to next level
			level++
			si.solutions[level] = newSolution
			updatedPattern := si.patterns[level].UpdateWithSolution(si.solutions[level])
			iter, err := si.db.GetIterator(si.ctx, updatedPattern)
			if err != nil {
				si.err = err
				return nil
			}
			si.iters[level] = iter
		} else {
			// Backtrack
			si.iters[level].Release()
			si.iters[level] = nil
			level--
		}
	}

	return nil
}

func (si *SolutionIterator) materialize(solution Solution, pattern *Pattern) Solution {
	tripleData := make(Solution)
	fields := []string{"subject", "predicate", "object"}
	for _, field := range fields {
		if v := pattern.GetVariable(field); v != nil {
			if val, ok := solution[v.Name]; ok {
				tripleData[field] = val
			}
		} else if val := pattern.GetConcreteValue(field); val != nil {
			tripleData[field] = val
		}
	}
	return tripleData
}

// Solution returns the current solution.
func (si *SolutionIterator) Solution() Solution {
	return si.current
}

// Close releases iterator resources.
func (si *SolutionIterator) Close() {
	if si.closed {
		return
	}
	si.closed = true
	for i, iter := range si.iters {
		if iter != nil {
			iter.Release()
			si.iters[i] = nil
		}
	}
}

// Error returns any error encountered during iteration.
func (si *SolutionIterator) Error() error {
	return si.err
}

// GetVectorScore extracts the vector similarity score from a solution.
// Returns 0 if no score was set (e.g., if VectorFilter wasn't used).
func GetVectorScore(sol Solution) float32 {
	scoreBytes, ok := sol["__vector_score__"]
	if !ok {
		return 0
	}
	scores := vector.BytesToVector(scoreBytes)
	if len(scores) == 0 {
		return 0
	}
	return scores[0]
}

// scoredSolution pairs a solution with its vector similarity score.
type scoredSolution struct {
	solution Solution
	score    float32
}

// applyVectorFilter filters and ranks solutions based on vector similarity.
func (db *DB) applyVectorFilter(ctx context.Context, solutions []Solution, vf *VectorFilter) ([]Solution, error) {
	if len(solutions) == 0 {
		return solutions, nil
	}

	// Get the query vector
	queryVec := vf.Query
	if queryVec == nil && vf.QueryText != "" {
		if db.options.Embedder == nil {
			return nil, ErrEmbedderRequired
		}
		var err error
		queryVec, err = db.options.Embedder.Embed(vf.QueryText)
		if err != nil {
			return nil, err
		}
	}

	if queryVec == nil {
		return solutions, nil // No query, return as-is
	}

	// Determine ID type
	idType := vf.IDType
	if idType == "" {
		idType = vector.IDTypeObject
	}

	// Score each solution based on vector similarity
	scored := make([]scoredSolution, 0, len(solutions))
	scoreCache := make(map[string]float32) // Cache scores by vector ID string

	for _, sol := range solutions {
		varValue, ok := sol[vf.Variable]
		if !ok {
			continue // Variable not bound in this solution
		}

		// Create vector ID for this value
		vecID := vector.MakeID(idType, varValue)
		vecIDStr := string(vecID)

		// Check if we've already scored this value
		if cachedScore, found := scoreCache[vecIDStr]; found {
			// Use the cached score for duplicate variable values
			scored = append(scored, scoredSolution{
				solution: sol,
				score:    cachedScore,
			})
			continue
		}

		// Look up vector for this value
		vec, err := db.options.VectorIndex.Get(vecID)
		if err != nil {
			// Vector not found for this value - assign score 0
			scoreCache[vecIDStr] = 0
			scored = append(scored, scoredSolution{
				solution: sol,
				score:    0,
			})
			continue
		}

		// Compute similarity and normalize to [0, 1] range
		// Use cosine distance, then normalize using the same function as Index.Search
		distance := vector.Cosine(queryVec, vec)
		normalizedScore := vector.NormalizeScore(distance)

		// Cache and store the score
		scoreCache[vecIDStr] = normalizedScore
		scored = append(scored, scoredSolution{
			solution: sol,
			score:    normalizedScore,
		})
	}

	// Apply minimum score filter
	if vf.MinScore > 0 {
		filtered := make([]scoredSolution, 0, len(scored))
		for _, s := range scored {
			if s.score >= vf.MinScore {
				filtered = append(filtered, s)
			}
		}
		scored = filtered
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Apply TopK limit
	if vf.TopK > 0 && len(scored) > vf.TopK {
		scored = scored[:vf.TopK]
	}

	// Extract solutions, adding score to each
	result := make([]Solution, len(scored))
	for i, s := range scored {
		// Clone solution and add score
		result[i] = s.solution.Clone()
		// Store score as special key (using float bytes)
		result[i]["__vector_score__"] = vector.VectorToBytes([]float32{s.score})
	}

	return result, nil
}
