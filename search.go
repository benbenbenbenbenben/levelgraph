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
func (db *DB) Search(patterns []*Pattern, opts *SearchOptions) ([]Solution, error) {
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

	// Apply limit
	if opts.Limit > 0 && opts.Limit < len(solutions) {
		solutions = solutions[:opts.Limit]
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
func (db *DB) SearchIterator(patterns []*Pattern, opts *SearchOptions) (*SolutionIterator, error) {
	// For now, we collect all results and iterate over them
	// A more sophisticated implementation would stream results
	solutions, err := db.Search(patterns, opts)
	if err != nil {
		return nil, err
	}

	return &SolutionIterator{
		solutions: solutions,
		index:     -1,
	}, nil
}

// SolutionIterator iterates over search solutions.
type SolutionIterator struct {
	solutions []Solution
	index     int
}

// Next advances to the next solution.
func (si *SolutionIterator) Next() bool {
	si.index++
	return si.index < len(si.solutions)
}

// Solution returns the current solution.
func (si *SolutionIterator) Solution() Solution {
	if si.index < 0 || si.index >= len(si.solutions) {
		return nil
	}
	return si.solutions[si.index]
}

// Close releases iterator resources.
func (si *SolutionIterator) Close() {
	si.solutions = nil
}
