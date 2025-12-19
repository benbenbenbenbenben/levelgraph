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
)

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
