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

package graph

import "bytes"

// Variable represents a named query placeholder used in pattern matching.
// When used in a Pattern, a Variable will match any value and capture it
// into the resulting Solution map under its name.
//
// Despite its name, this is conceptually a "binding" or "placeholder" rather
// than a mutable variable. Once bound in a solution, the value is immutable.
// The name follows the original JavaScript LevelGraph API for compatibility.
//
// For a more descriptive API, consider using the type-safe PatternValue
// with Binding("name") instead. See pattern_typed.go for details.
type Variable struct {
	// Name is the identifier for this binding in the Solution map.
	Name string `json:"name"`
}

// Var is an alias for Variable, providing a shorter name for those who prefer it.
// Usage: pattern.Subject = levelgraph.Var("x")
type Var = Variable

// V creates a new Variable with the given name.
// This is the primary constructor and mimics the db.v("name") pattern from JS.
//
// Example:
//
//	pattern := &Pattern{
//	    Subject:   []byte("alice"),
//	    Predicate: []byte("knows"),
//	    Object:    V("friend"),  // Captures matched object as "friend"
//	}
func V(name string) *Variable {
	return &Variable{Name: name}
}

// Bind attempts to bind this variable to a value within a solution.
// If the variable is already bound to a different value, it returns nil (no match).
// If the variable is unbound or bound to the same value, it returns the updated solution.
func (v *Variable) Bind(solution Solution, value []byte) Solution {
	if !v.IsBindable(solution, value) {
		return nil
	}

	// Create a new solution with the binding
	newSolution := make(Solution, len(solution)+1)
	for k, val := range solution {
		newSolution[k] = val
	}
	newSolution[v.Name] = value
	return newSolution
}

// IsBound returns true if this variable is already bound in the solution.
func (v *Variable) IsBound(solution Solution) bool {
	_, exists := solution[v.Name]
	return exists
}

// IsBindable returns true if this variable can be bound to the given value.
// A variable is bindable if it's either unbound, or already bound to the same value.
func (v *Variable) IsBindable(solution Solution, value []byte) bool {
	existing, exists := solution[v.Name]
	if !exists {
		return true
	}
	return bytes.Equal(existing, value)
}

// GetValue returns the bound value if this variable is bound, or nil if unbound.
func (v *Variable) GetValue(solution Solution) []byte {
	return solution[v.Name]
}

// Solution represents a mapping of variable names to their bound values.
// This is the result of a successful pattern match or search.
type Solution map[string][]byte

// Clone creates a deep copy of the solution.
func (s Solution) Clone() Solution {
	if s == nil {
		return nil
	}
	clone := make(Solution, len(s))
	for k, v := range s {
		clone[k] = append([]byte(nil), v...)
	}
	return clone
}

// Equal returns true if two solutions have identical bindings.
func (s Solution) Equal(other Solution) bool {
	if len(s) != len(other) {
		return false
	}
	for k, v := range s {
		if !bytes.Equal(v, other[k]) {
			return false
		}
	}
	return true
}

// IsVariable checks if the given interface{} is a *Variable.
func IsVariable(v interface{}) bool {
	_, ok := v.(*Variable)
	return ok
}

// AsVariable converts an interface{} to *Variable if possible.
func AsVariable(v interface{}) (*Variable, bool) {
	variable, ok := v.(*Variable)
	return variable, ok
}
