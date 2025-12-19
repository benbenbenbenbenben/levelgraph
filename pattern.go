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

// Pattern represents a query pattern that can match triples.
// Each field can be:
//   - nil: matches any value (wildcard)
//   - []byte: matches exactly that value
//   - *Variable: binds matched value to the variable name
type Pattern struct {
	// Subject can be nil (wildcard), []byte (exact match), or *Variable
	Subject interface{}
	// Predicate can be nil (wildcard), []byte (exact match), or *Variable
	Predicate interface{}
	// Object can be nil (wildcard), []byte (exact match), or *Variable
	Object interface{}

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
func normalizePatternValue(v interface{}) interface{} {
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
	case *Variable:
		return val
	case bool:
		// Handle false as a valid value (like JS version)
		if !val {
			return []byte("false")
		}
		return []byte("true")
	default:
		return nil
	}
}

// GetConcreteValue returns the concrete []byte value for a field, or nil if the field
// is a wildcard or variable.
func (p *Pattern) GetConcreteValue(field string) []byte {
	var v interface{}
	switch field {
	case "subject":
		v = p.Subject
	case "predicate":
		v = p.Predicate
	case "object":
		v = p.Object
	default:
		return nil
	}

	if v == nil {
		return nil
	}
	if b, ok := v.([]byte); ok {
		return b
	}
	return nil
}

// GetVariable returns the Variable for a field, or nil if it's not a variable.
func (p *Pattern) GetVariable(field string) *Variable {
	var v interface{}
	switch field {
	case "subject":
		v = p.Subject
	case "predicate":
		v = p.Predicate
	case "object":
		v = p.Object
	default:
		return nil
	}

	if variable, ok := v.(*Variable); ok {
		return variable
	}
	return nil
}

// HasVariable returns true if any field contains a variable.
func (p *Pattern) HasVariable() bool {
	return IsVariable(p.Subject) || IsVariable(p.Predicate) || IsVariable(p.Object)
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
		if !bytesEqual(s, triple.Subject) {
			return false
		}
	}
	if pr := p.GetConcreteValue("predicate"); pr != nil {
		if !bytesEqual(pr, triple.Predicate) {
			return false
		}
	}
	if o := p.GetConcreteValue("object"); o != nil {
		if !bytesEqual(o, triple.Object) {
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
			newPattern.Subject = val
		}
	}
	if v := p.GetVariable("predicate"); v != nil {
		if val, ok := solution[v.Name]; ok {
			newPattern.Predicate = val
		}
	}
	if v := p.GetVariable("object"); v != nil {
		if val, ok := solution[v.Name]; ok {
			newPattern.Object = val
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
		if !bytesEqual(s, triple.Subject) {
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
		if !bytesEqual(pr, triple.Predicate) {
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
		if !bytesEqual(o, triple.Object) {
			return nil
		}
	}

	return newSolution
}
