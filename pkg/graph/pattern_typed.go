// Copyright (c) 2013-2025 Matteo Collina and LevelGraph Contributors
// Copyright (c) 2025 LevelGraph Go Contributors
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

// PatternValue represents a type-safe pattern field value.
// This is the recommended approach for Go 1.18+ to avoid interface{} in patterns.
//
// Use the constructor functions to create values:
//   - Exact([]byte) or ExactString(string) for concrete matches
//   - Wildcard() for matching any value
//   - Binding(name) for capturing values into named variables
//
// Example usage:
//
//	pattern := &TypedPattern{
//	    Subject:   Exact([]byte("alice")),
//	    Predicate: ExactString("knows"),
//	    Object:    Binding("friend"),
//	}
type PatternValue struct {
	kind     patternValueKind
	data     []byte
	variable *Variable
}

type patternValueKind int

const (
	patternValueWildcard patternValueKind = iota
	patternValueExact
	patternValueBinding
)

// Wildcard creates a PatternValue that matches any value.
// Equivalent to nil in the original Pattern interface{} fields.
func Wildcard() PatternValue {
	return PatternValue{kind: patternValueWildcard}
}

// Exact creates a PatternValue that matches exactly the given bytes.
func Exact(data []byte) PatternValue {
	return PatternValue{kind: patternValueExact, data: data}
}

// ExactString creates a PatternValue that matches exactly the given string.
// This is a convenience wrapper around Exact.
func ExactString(s string) PatternValue {
	return PatternValue{kind: patternValueExact, data: []byte(s)}
}

// Binding creates a PatternValue that binds the matched value to a named variable.
// This is the type-safe equivalent of using V("name") in pattern fields.
func Binding(name string) PatternValue {
	return PatternValue{kind: patternValueBinding, variable: V(name)}
}

// IsWildcard returns true if this value matches anything.
func (pv PatternValue) IsWildcard() bool {
	return pv.kind == patternValueWildcard
}

// IsExact returns true if this value matches a specific byte sequence.
func (pv PatternValue) IsExact() bool {
	return pv.kind == patternValueExact
}

// IsBinding returns true if this value captures into a variable.
func (pv PatternValue) IsBinding() bool {
	return pv.kind == patternValueBinding
}

// Data returns the exact data if this is an exact match, or nil otherwise.
func (pv PatternValue) Data() []byte {
	if pv.kind == patternValueExact {
		return pv.data
	}
	return nil
}

// VariableName returns the variable name if this is a binding, or empty string otherwise.
func (pv PatternValue) VariableName() string {
	if pv.kind == patternValueBinding && pv.variable != nil {
		return pv.variable.Name
	}
	return ""
}

// ToInterface converts the PatternValue to the interface{} representation
// used by the original Pattern struct. This enables interoperability.
func (pv PatternValue) ToInterface() interface{} {
	switch pv.kind {
	case patternValueWildcard:
		return nil
	case patternValueExact:
		return pv.data
	case patternValueBinding:
		return pv.variable
	default:
		return nil
	}
}

// TypedPattern is a type-safe alternative to Pattern that uses PatternValue
// instead of interface{} for the triple fields.
//
// This is the recommended pattern type for new code in Go 1.18+.
// Use ToPattern() to convert to the standard Pattern for use with DB methods.
type TypedPattern struct {
	Subject   PatternValue
	Predicate PatternValue
	Object    PatternValue

	// Filter is an optional function to filter results
	Filter func(*Triple) bool

	// Limit restricts the number of results (0 or negative means no limit)
	Limit int
	// Offset skips the first N results
	Offset int
	// Reverse iterates in reverse lexicographical order
	Reverse bool
}

// NewTypedPattern creates a TypedPattern with the given field values.
func NewTypedPattern(subject, predicate, object PatternValue) *TypedPattern {
	return &TypedPattern{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
	}
}

// ToPattern converts a TypedPattern to the standard Pattern type.
// This enables use with all existing DB methods.
func (tp *TypedPattern) ToPattern() *Pattern {
	return &Pattern{
		Subject:   tp.Subject.ToInterface(),
		Predicate: tp.Predicate.ToInterface(),
		Object:    tp.Object.ToInterface(),
		Filter:    tp.Filter,
		Limit:     tp.Limit,
		Offset:    tp.Offset,
		Reverse:   tp.Reverse,
	}
}

// PatternFromTyped is an alias for TypedPattern.ToPattern() for convenience.
func PatternFromTyped(tp *TypedPattern) *Pattern {
	return tp.ToPattern()
}
