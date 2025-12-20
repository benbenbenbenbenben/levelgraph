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

import (
	"bytes"
	"strconv"
)

// PatternValue represents a type-safe pattern field value.
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
func Wildcard() PatternValue {
	return PatternValue{kind: patternValueWildcard}
}

// Exact creates a PatternValue that matches exactly the given bytes.
func Exact(data []byte) PatternValue {
	return PatternValue{kind: patternValueExact, data: data}
}

// ExactString creates a PatternValue that matches exactly the given string.
func ExactString(s string) PatternValue {
	return PatternValue{kind: patternValueExact, data: []byte(s)}
}

// Binding creates a PatternValue that binds the matched value to a named variable.
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

// Pattern represents a query pattern that can match triples.
// It uses PatternValue for type-safe field matching.
type Pattern struct {
	// Subject defines the match criteria for the triple subject
	Subject PatternValue
	// Predicate defines the match criteria for the triple predicate
	Predicate PatternValue
	// Object defines the match criteria for the triple object
	Object PatternValue

	// Filter is an optional function to filter results
	Filter func(*Triple) bool

	// Limit restricts the number of results (0 or negative means no limit)
	Limit int
	// Offset skips the first N results
	Offset int
	// Reverse iterates in reverse lexicographical order
	Reverse bool
}

// NewPattern creates a new pattern from interface values.
// Values can be nil, []byte, string (converted to []byte), or *Variable.
func NewPattern(subject, predicate, object interface{}) *Pattern {
	return &Pattern{
		Subject:   normalizePatternValue(subject),
		Predicate: normalizePatternValue(predicate),
		Object:    normalizePatternValue(object),
	}
}

// normalizePatternValue converts various input types to the internal representation.
func normalizePatternValue(v interface{}) PatternValue {
	if v == nil {
		return Wildcard()
	}
	switch val := v.(type) {
	case PatternValue:
		return val
	case []byte:
		if len(val) == 0 {
			return Wildcard()
		}
		return Exact(val)
	case string:
		if val == "" {
			return Wildcard()
		}
		return ExactString(val)
	case *Variable:
		return PatternValue{kind: patternValueBinding, variable: val}
	case bool:
		// Convert boolean to its string representation using strconv for clarity
		return ExactString(strconv.FormatBool(val))
	default:
		return Wildcard()
	}
}

// GetConcreteValue returns the concrete []byte value for a field, or nil if the field
// is a wildcard or variable.
func (p *Pattern) GetConcreteValue(field string) []byte {
	var pv PatternValue
	switch field {
	case "subject":
		pv = p.Subject
	case "predicate":
		pv = p.Predicate
	case "object":
		pv = p.Object
	default:
		return nil
	}

	return pv.Data()
}

// GetVariable returns the Variable for a field, or nil if it's not a variable.
func (p *Pattern) GetVariable(field string) *Variable {
	var pv PatternValue
	switch field {
	case "subject":
		pv = p.Subject
	case "predicate":
		pv = p.Predicate
	case "object":
		pv = p.Object
	default:
		return nil
	}

	if pv.IsBinding() {
		return pv.variable
	}
	return nil
}

// HasVariable returns true if any field contains a variable.
func (p *Pattern) HasVariable() bool {
	return p.Subject.IsBinding() || p.Predicate.IsBinding() || p.Object.IsBinding()
}

// ConcreteFields returns the names of fields that have concrete (non-variable, non-nil) values.
func (p *Pattern) ConcreteFields() []string {
	var fields []string
	if p.GetConcreteValue("subject") != nil {
		fields = append(fields, "subject")
	}
	if p.GetConcreteValue("predicate") != nil {
		fields = append(fields, "predicate")
	}
	if p.GetConcreteValue("object") != nil {
		fields = append(fields, "object")
	}
	return fields
}

// VariableFields returns a map of field names to their Variable objects.
func (p *Pattern) VariableFields() map[string]*Variable {
	result := make(map[string]*Variable)
	if v := p.GetVariable("subject"); v != nil {
		result["subject"] = v
	}
	if v := p.GetVariable("predicate"); v != nil {
		result["predicate"] = v
	}
	if v := p.GetVariable("object"); v != nil {
		result["object"] = v
	}
	return result
}

// ToTriple converts a pattern with all concrete values to a Triple.
// Returns nil if any field is nil or a variable.
func (p *Pattern) ToTriple() *Triple {
	s := p.GetConcreteValue("subject")
	pr := p.GetConcreteValue("predicate")
	o := p.GetConcreteValue("object")

	if s == nil || pr == nil || o == nil {
		return nil
	}

	return &Triple{
		Subject:   s,
		Predicate: pr,
		Object:    o,
	}
}

// Matches returns true if the given triple matches this pattern.
func (p *Pattern) Matches(triple *Triple) bool {
	if s := p.GetConcreteValue("subject"); s != nil {
		if !bytes.Equal(s, triple.Subject) {
			return false
		}
	}
	if pr := p.GetConcreteValue("predicate"); pr != nil {
		if !bytes.Equal(pr, triple.Predicate) {
			return false
		}
	}
	if o := p.GetConcreteValue("object"); o != nil {
		if !bytes.Equal(o, triple.Object) {
			return false
		}
	}
	return true
}

// UpdateWithSolution returns a new pattern with variables replaced by their bound values.
func (p *Pattern) UpdateWithSolution(solution Solution) *Pattern {
	newPattern := &Pattern{
		Subject:   p.Subject,
		Predicate: p.Predicate,
		Object:    p.Object,
		Filter:    p.Filter,
		Limit:     p.Limit,
		Offset:    p.Offset,
		Reverse:   p.Reverse,
	}

	// Replace variables with bound values
	if v := p.GetVariable("subject"); v != nil {
		if val, ok := solution[v.Name]; ok {
			newPattern.Subject = Exact(val)
		}
	}
	if v := p.GetVariable("predicate"); v != nil {
		if val, ok := solution[v.Name]; ok {
			newPattern.Predicate = Exact(val)
		}
	}
	if v := p.GetVariable("object"); v != nil {
		if val, ok := solution[v.Name]; ok {
			newPattern.Object = Exact(val)
		}
	}

	return newPattern
}

// BindTriple attempts to bind a triple's values to variables in this pattern.
// Returns the updated solution if successful, or nil if the triple doesn't match.
func (p *Pattern) BindTriple(solution Solution, triple *Triple) Solution {
	newSolution := solution.Clone()
	if newSolution == nil {
		newSolution = make(Solution)
	}

	// Check and bind subject
	if v := p.GetVariable("subject"); v != nil {
		newSolution = v.Bind(newSolution, triple.Subject)
		if newSolution == nil {
			return nil
		}
	} else if s := p.GetConcreteValue("subject"); s != nil {
		if !bytes.Equal(s, triple.Subject) {
			return nil
		}
	}

	// Check and bind predicate
	if v := p.GetVariable("predicate"); v != nil {
		newSolution = v.Bind(newSolution, triple.Predicate)
		if newSolution == nil {
			return nil
		}
	} else if pr := p.GetConcreteValue("predicate"); pr != nil {
		if !bytes.Equal(pr, triple.Predicate) {
			return nil
		}
	}

	// Check and bind object
	if v := p.GetVariable("object"); v != nil {
		newSolution = v.Bind(newSolution, triple.Object)
		if newSolution == nil {
			return nil
		}
	} else if o := p.GetConcreteValue("object"); o != nil {
		if !bytes.Equal(o, triple.Object) {
			return nil
		}
	}

	return newSolution
}

// BindTripleFast is an optimized version of BindTriple that uses shallow cloning.
// It creates a new solution map but shares byte slice references with the input.
// This is safe because triple values from the database are not modified.
func (p *Pattern) BindTripleFast(solution Solution, triple *Triple) Solution {
	// Use shallow clone - we're only adding new bindings, not modifying existing values
	newSolution := solution.ShallowClone()
	if newSolution == nil {
		newSolution = make(Solution)
	}

	// Check and bind subject
	if p.Subject.IsBinding() {
		v := p.Subject.variable
		if !v.BindInPlace(newSolution, triple.Subject) {
			return nil
		}
	} else if p.Subject.IsExact() {
		if !bytes.Equal(p.Subject.data, triple.Subject) {
			return nil
		}
	}

	// Check and bind predicate
	if p.Predicate.IsBinding() {
		v := p.Predicate.variable
		if !v.BindInPlace(newSolution, triple.Predicate) {
			return nil
		}
	} else if p.Predicate.IsExact() {
		if !bytes.Equal(p.Predicate.data, triple.Predicate) {
			return nil
		}
	}

	// Check and bind object
	if p.Object.IsBinding() {
		v := p.Object.variable
		if !v.BindInPlace(newSolution, triple.Object) {
			return nil
		}
	} else if p.Object.IsExact() {
		if !bytes.Equal(p.Object.data, triple.Object) {
			return nil
		}
	}

	return newSolution
}
