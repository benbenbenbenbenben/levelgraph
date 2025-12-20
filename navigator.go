// Copyright (c) 2013-2024 Matteo Collina and LevelGraph Contributors
// Copyright (c) 2024 LevelGraph Go Contributors
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
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
	"fmt"

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
)

// Navigator provides a fluent API for traversing the graph.
// It allows building queries by following edges in and out.
//
// Example usage:
//
//	nav := db.Nav(ctx, []byte("alice"))
//	solutions, err := nav.ArchOut("knows").ArchOut("likes").Solutions()
//
// This finds all things liked by people that alice knows.
type Navigator struct {
	ctx             context.Context
	db              *DB
	conditions      []*graph.Pattern
	initialSolution graph.Solution
	lastElement     interface{} // either []byte or *graph.Variable
	varCounter      int
}

// Nav creates a new Navigator starting from the given vertex.
// If start is nil, a new variable is created as the starting point.
func (db *DB) Nav(ctx context.Context, start interface{}) *Navigator {
	nav := &Navigator{
		ctx:             ctx,
		db:              db,
		conditions:      make([]*graph.Pattern, 0),
		initialSolution: make(graph.Solution),
		varCounter:      0,
	}
	nav.Go(start)
	return nav
}

// nextVar generates the next anonymous variable for this navigator.
func (nav *Navigator) nextVar() *graph.Variable {
	v := graph.V(fmt.Sprintf("x%d", nav.varCounter))
	nav.varCounter++
	return v
}

// Go moves the navigator to a new vertex.
// If vertex is nil, a new variable is created.
// vertex can be []byte, string (converted to []byte), or *graph.Variable.
func (nav *Navigator) Go(vertex interface{}) *Navigator {
	if vertex == nil {
		nav.lastElement = nav.nextVar()
	} else {
		switch v := vertex.(type) {
		case []byte:
			nav.lastElement = v
		case string:
			nav.lastElement = []byte(v)
		case *graph.Variable:
			nav.lastElement = v
		default:
			nav.lastElement = nav.nextVar()
		}
	}
	return nav
}

// ArchOut follows an outgoing edge with the given predicate.
// The current position becomes the subject, and navigates to the object.
func (nav *Navigator) ArchOut(predicate interface{}) *Navigator {
	pred := normalizeValue(predicate)
	newVar := nav.nextVar()

	pattern := &graph.Pattern{
		Subject:   nav.lastElement,
		Predicate: pred,
		Object:    newVar,
	}

	nav.conditions = append(nav.conditions, pattern)
	nav.lastElement = newVar
	return nav
}

// ArchIn follows an incoming edge with the given predicate.
// The current position becomes the object, and navigates to the subject.
func (nav *Navigator) ArchIn(predicate interface{}) *Navigator {
	pred := normalizeValue(predicate)
	newVar := nav.nextVar()

	pattern := &graph.Pattern{
		Subject:   newVar,
		Predicate: pred,
		Object:    nav.lastElement,
	}

	nav.conditions = append(nav.conditions, pattern)
	nav.lastElement = newVar
	return nav
}

// As names the current position with the given variable name.
// This allows referencing the position later in the query.
func (nav *Navigator) As(name string) *Navigator {
	if v, ok := nav.lastElement.(*graph.Variable); ok {
		v.Name = name
	}
	return nav
}

// Bind binds the current position's variable to a concrete value.
// This is used to constrain the search.
func (nav *Navigator) Bind(value interface{}) *Navigator {
	if v, ok := nav.lastElement.(*graph.Variable); ok {
		val := normalizeValue(value)
		if val != nil {
			nav.initialSolution[v.Name] = val
		}
	}
	return nav
}

// Solutions executes the navigation query and returns all solutions.
// Each solution is a map of variable names to their bound values.
func (nav *Navigator) Solutions() ([]graph.Solution, error) {
	if len(nav.conditions) == 0 {
		// No conditions means return the initial solution
		return []graph.Solution{nav.initialSolution}, nil
	}

	// Pass initial solution to search - patterns will be updated with bound values,
	// and the initial solution will be included in results
	return nav.db.Search(nav.ctx, nav.conditions, &SearchOptions{
		InitialSolution: nav.initialSolution,
	})
}

// Values returns unique values for the last navigated position.
// This is useful for getting distinct nodes at the end of a traversal.
func (nav *Navigator) Values() ([][]byte, error) {
	solutions, err := nav.Solutions()
	if err != nil {
		return nil, err
	}

	// Get the variable name of the last element
	var varName string
	if v, ok := nav.lastElement.(*graph.Variable); ok {
		varName = v.Name
	} else {
		// Last element is a concrete value, return it
		if b, ok := nav.lastElement.([]byte); ok {
			return [][]byte{b}, nil
		}
		return nil, nil
	}

	// Collect unique values
	seen := make(map[string]bool)
	var result [][]byte

	for _, sol := range solutions {
		if val, ok := sol[varName]; ok {
			key := string(val)
			if !seen[key] {
				seen[key] = true
				result = append(result, val)
			}
		}
	}

	return result, nil
}

// Triples executes the query and materializes results into triples.
// The pattern specifies how to construct the result triples from solutions.
func (nav *Navigator) Triples(pattern *graph.Pattern) ([]*graph.Triple, error) {
	if len(nav.conditions) == 0 {
		return nil, nil
	}

	solutions, err := nav.db.Search(nav.ctx, nav.conditions, &SearchOptions{
		InitialSolution: nav.initialSolution,
		Materialized:    pattern,
	})
	if err != nil {
		return nil, err
	}

	// Convert solutions to triples
	var result []*graph.Triple
	for _, sol := range solutions {
		triple := &graph.Triple{}
		if s, ok := sol["subject"]; ok {
			triple.Subject = s
		}
		if p, ok := sol["predicate"]; ok {
			triple.Predicate = p
		}
		if o, ok := sol["object"]; ok {
			triple.Object = o
		}
		if triple.Subject != nil && triple.Predicate != nil && triple.Object != nil {
			result = append(result, triple)
		}
	}

	return result, nil
}

// normalizeValue converts various input types to []byte.
func normalizeValue(v interface{}) []byte {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		if len(val) == 0 {
			return nil
		}
		return val
	case string:
		if val == "" {
			return nil
		}
		return []byte(val)
	default:
		return nil
	}
}

// Count returns the number of solutions without materializing all of them.
func (nav *Navigator) Count() (int, error) {
	solutions, err := nav.Solutions()
	if err != nil {
		return 0, err
	}
	return len(solutions), nil
}

// First returns the first solution, or nil if none found.
func (nav *Navigator) First() (graph.Solution, error) {
	if len(nav.conditions) == 0 {
		return nav.initialSolution, nil
	}

	solutions, err := nav.db.Search(nav.ctx, nav.conditions, &SearchOptions{
		InitialSolution: nav.initialSolution,
		Limit:           1,
	})
	if err != nil {
		return nil, err
	}

	if len(solutions) == 0 {
		return nil, nil
	}

	return solutions[0], nil
}

// Exists returns true if at least one solution exists.
func (nav *Navigator) Exists() (bool, error) {
	sol, err := nav.First()
	return sol != nil, err
}

// Clone creates a copy of this navigator that can be modified independently.
func (nav *Navigator) Clone() *Navigator {
	newNav := &Navigator{
		ctx:             nav.ctx,
		db:              nav.db,
		conditions:      make([]*graph.Pattern, len(nav.conditions)),
		initialSolution: make(graph.Solution),
		lastElement:     nav.lastElement,
		varCounter:      nav.varCounter,
	}

	copy(newNav.conditions, nav.conditions)

	for k, v := range nav.initialSolution {
		newNav.initialSolution[k] = v
	}

	return newNav
}

// Filter adds a filter function to the last condition.
// The filter is applied to each matching triple.
func (nav *Navigator) Filter(fn func(*graph.Triple) bool) *Navigator {
	if len(nav.conditions) > 0 {
		nav.conditions[len(nav.conditions)-1].Filter = fn
	}
	return nav
}

// Where adds a custom pattern condition to the navigator.
func (nav *Navigator) Where(pattern *graph.Pattern) *Navigator {
	nav.conditions = append(nav.conditions, pattern)
	return nav
}
