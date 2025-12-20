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

package index

import (
	"bytes"
	"testing"

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
)

func TestEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{"nil input", nil, nil},
		{"empty input", []byte{}, []byte{}},
		{"no special chars", []byte("hello"), []byte("hello")},
		{"with colon", []byte("hello:world"), []byte(`hello\:world`)},
		{"with backslash", []byte(`hello\world`), []byte(`hello\\world`)},
		{"with both", []byte(`hello:wo\rld`), []byte(`hello\:wo\\rld`)},
		{"multiple colons", []byte("a:b:c"), []byte(`a\:b\:c`)},
		{"consecutive special", []byte(`:\:`), []byte(`\:\\\:`)},
		{"only colon", []byte(":"), []byte(`\:`)},
		{"only backslash", []byte(`\`), []byte(`\\`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Escape(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Escape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUnescape(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{"nil input", nil, nil},
		{"empty input", []byte{}, []byte{}},
		{"no escapes", []byte("hello"), []byte("hello")},
		{"escaped colon", []byte(`hello\:world`), []byte("hello:world")},
		{"escaped backslash", []byte(`hello\\world`), []byte(`hello\world`)},
		{"escaped both", []byte(`hello\:wo\\rld`), []byte(`hello:wo\rld`)},
		{"multiple escaped colons", []byte(`a\:b\:c`), []byte("a:b:c")},
		{"consecutive escaped", []byte(`\:\\\:`), []byte(`:\:`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unescape(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Unescape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	testCases := [][]byte{
		[]byte("simple"),
		[]byte("with:colon"),
		[]byte(`with\backslash`),
		[]byte(`both:and\here`),
		[]byte(":::"),
		[]byte(`\\\`),
		[]byte(`mixed:\:stuff\\here`),
		[]byte(""),
	}

	for _, input := range testCases {
		escaped := Escape(input)
		unescaped := Unescape(escaped)
		if !bytes.Equal(input, unescaped) {
			t.Errorf("Round trip failed for %q: escaped=%q, unescaped=%q", input, escaped, unescaped)
		}
	}
}

func TestGenKey(t *testing.T) {
	triple := graph.NewTripleFromStrings("alice", "knows", "bob")

	tests := []struct {
		index    IndexName
		expected string
	}{
		{IndexSPO, "spo::alice::knows::bob"},
		{IndexSOP, "sop::alice::bob::knows"},
		{IndexPOS, "pos::knows::bob::alice"},
		{IndexPSO, "pso::knows::alice::bob"},
		{IndexOPS, "ops::bob::knows::alice"},
		{IndexOSP, "osp::bob::alice::knows"},
	}

	for _, tt := range tests {
		t.Run(string(tt.index), func(t *testing.T) {
			result := GenKey(tt.index, triple)
			if string(result) != tt.expected {
				t.Errorf("GenKey(%s) = %q, want %q", tt.index, result, tt.expected)
			}
		})
	}
}

func TestGenKeyWithEscaping(t *testing.T) {
	triple := &graph.Triple{
		Subject:   []byte("alice:admin"),
		Predicate: []byte("has:role"),
		Object:    []byte(`value\with\backslash`),
	}

	key := GenKey(IndexSPO, triple)
	expected := `spo::alice\:admin::has\:role::value\\with\\backslash`
	if string(key) != expected {
		t.Errorf("GenKey with escaping = %q, want %q", key, expected)
	}
}

func TestGenKeys(t *testing.T) {
	triple := graph.NewTripleFromStrings("s", "p", "o")
	keys := GenKeys(triple)

	if len(keys) != 6 {
		t.Errorf("GenKeys returned %d keys, want 6", len(keys))
	}

	// Verify each index is represented
	indexNames := make(map[string]bool)
	for _, key := range keys {
		parts := bytes.SplitN(key, KeySeparator, 2)
		indexNames[string(parts[0])] = true
	}

	for _, idx := range AllIndexes {
		if !indexNames[string(idx)] {
			t.Errorf("Missing key for index %s", idx)
		}
	}
}

func TestGenKeyFromPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  *graph.Pattern
		index    IndexName
		expected string
	}{
		{
			name:     "full pattern SPO",
			pattern:  graph.NewPattern("alice", "knows", "bob"),
			index:    IndexSPO,
			expected: "spo::alice::knows::bob",
		},
		{
			name:     "subject only",
			pattern:  graph.NewPattern("alice", nil, nil),
			index:    IndexSPO,
			expected: "spo::alice::",
		},
		{
			name:     "subject and predicate",
			pattern:  graph.NewPattern("alice", "knows", nil),
			index:    IndexSPO,
			expected: "spo::alice::knows::",
		},
		{
			name:     "predicate only - use POS",
			pattern:  graph.NewPattern(nil, "knows", nil),
			index:    IndexPOS,
			expected: "pos::knows::",
		},
		{
			name:     "object only - use OPS",
			pattern:  graph.NewPattern(nil, nil, "bob"),
			index:    IndexOPS,
			expected: "ops::bob::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenKeyFromPattern(tt.index, tt.pattern)
			if string(result) != tt.expected {
				t.Errorf("GenKeyFromPattern(%s, %v) = %q, want %q", tt.index, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestGenKeyWithUpperBound(t *testing.T) {
	tests := []struct {
		name    string
		pattern *graph.Pattern
		index   IndexName
	}{
		{
			name:    "partial pattern",
			pattern: graph.NewPattern("alice", nil, nil),
			index:   IndexSPO,
		},
		{
			name:    "full pattern",
			pattern: graph.NewPattern("alice", "knows", "bob"),
			index:   IndexSPO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lower := GenKeyFromPattern(tt.index, tt.pattern)
			upper := GenKeyWithUpperBound(tt.index, tt.pattern)

			// Upper bound should always be greater than or equal to lower bound
			if bytes.Compare(upper, lower) < 0 {
				t.Errorf("Upper bound %q is less than lower bound %q", upper, lower)
			}

			// Upper bound should end with 0xFF byte(s)
			if upper[len(upper)-1] != 0xFF {
				t.Errorf("Upper bound %q doesn't end with 0xFF", upper)
			}
		})
	}
}

func TestPossibleIndexes(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		expected []IndexName
	}{
		{
			name:     "subject only",
			fields:   []string{"subject"},
			expected: []IndexName{IndexSOP, IndexSPO},
		},
		{
			name:     "predicate only",
			fields:   []string{"predicate"},
			expected: []IndexName{IndexPOS, IndexPSO},
		},
		{
			name:     "object only",
			fields:   []string{"object"},
			expected: []IndexName{IndexOPS, IndexOSP},
		},
		{
			name:     "subject and predicate",
			fields:   []string{"subject", "predicate"},
			expected: []IndexName{IndexSPO},
		},
		{
			name:     "subject and object",
			fields:   []string{"subject", "object"},
			expected: []IndexName{IndexSOP},
		},
		{
			name:     "predicate and object",
			fields:   []string{"predicate", "object"},
			expected: []IndexName{IndexPOS},
		},
		{
			name:     "no fields (wildcard)",
			fields:   []string{},
			expected: []IndexName{IndexOPS, IndexOSP, IndexPOS, IndexPSO, IndexSOP, IndexSPO},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PossibleIndexes(tt.fields)
			if len(result) != len(tt.expected) {
				t.Errorf("PossibleIndexes(%v) returned %d indexes, want %d: got %v, want %v",
					tt.fields, len(result), len(tt.expected), result, tt.expected)
				return
			}
			for i, idx := range result {
				if idx != tt.expected[i] {
					t.Errorf("PossibleIndexes(%v)[%d] = %s, want %s", tt.fields, i, idx, tt.expected[i])
				}
			}
		})
	}
}

func TestFindIndex(t *testing.T) {
	tests := []struct {
		name      string
		fields    []string
		preferred IndexName
		expected  IndexName
	}{
		{
			name:      "subject with no preference",
			fields:    []string{"subject"},
			preferred: "",
			expected:  IndexSOP, // First alphabetically among SOP, SPO
		},
		{
			name:      "subject with valid preference",
			fields:    []string{"subject"},
			preferred: IndexSPO,
			expected:  IndexSPO,
		},
		{
			name:      "subject with invalid preference",
			fields:    []string{"subject"},
			preferred: IndexPOS, // Not valid for subject-only query
			expected:  IndexSOP, // Falls back to first valid
		},
		{
			name:      "no fields defaults to SPO",
			fields:    []string{},
			preferred: "",
			expected:  IndexOPS, // First alphabetically when all are valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindIndex(tt.fields, tt.preferred)
			if result != tt.expected {
				t.Errorf("FindIndex(%v, %s) = %s, want %s", tt.fields, tt.preferred, result, tt.expected)
			}
		})
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name           string
		key            []byte
		expectedIndex  IndexName
		expectedValues [][]byte
	}{
		{
			name:           "full SPO key",
			key:            []byte("spo::alice::knows::bob"),
			expectedIndex:  IndexSPO,
			expectedValues: [][]byte{[]byte("alice"), []byte("knows"), []byte("bob")},
		},
		{
			name:           "partial key",
			key:            []byte("spo::alice::"),
			expectedIndex:  IndexSPO,
			expectedValues: [][]byte{[]byte("alice")},
		},
		{
			name:           "key with escaped values",
			key:            []byte(`spo::alice\:admin::has\:role::value`),
			expectedIndex:  IndexSPO,
			expectedValues: [][]byte{[]byte("alice:admin"), []byte("has:role"), []byte("value")},
		},
		{
			name:           "empty key",
			key:            []byte(""),
			expectedIndex:  "",
			expectedValues: [][]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, values := ParseKey(tt.key)
			if index != tt.expectedIndex {
				t.Errorf("ParseKey(%q) index = %s, want %s", tt.key, index, tt.expectedIndex)
			}
			if len(values) != len(tt.expectedValues) {
				t.Errorf("ParseKey(%q) values length = %d, want %d", tt.key, len(values), len(tt.expectedValues))
				return
			}
			for i, v := range values {
				if !bytes.Equal(v, tt.expectedValues[i]) {
					t.Errorf("ParseKey(%q) values[%d] = %q, want %q", tt.key, i, v, tt.expectedValues[i])
				}
			}
		})
	}
}

func TestIndexDefs(t *testing.T) {
	// Verify all indexes are defined
	if len(IndexDefs) != 6 {
		t.Errorf("IndexDefs has %d entries, want 6", len(IndexDefs))
	}

	// Verify each index has exactly 3 fields
	for idx, def := range IndexDefs {
		if len(def) != 3 {
			t.Errorf("IndexDefs[%s] has %d fields, want 3", idx, len(def))
		}

		// Verify all fields are valid
		validFields := map[string]bool{"subject": true, "predicate": true, "object": true}
		for _, field := range def {
			if !validFields[field] {
				t.Errorf("IndexDefs[%s] contains invalid field %q", idx, field)
			}
		}
	}
}

func TestAllIndexes(t *testing.T) {
	if len(AllIndexes) != 6 {
		t.Errorf("AllIndexes has %d entries, want 6", len(AllIndexes))
	}

	// Verify all indexes in AllIndexes are in IndexDefs
	for _, idx := range AllIndexes {
		if _, ok := IndexDefs[idx]; !ok {
			t.Errorf("AllIndexes contains %s which is not in IndexDefs", idx)
		}
	}
}

func TestContainsField(t *testing.T) {
	fields := []string{"subject", "predicate"}

	if !containsField(fields, "subject") {
		t.Error("containsField should return true for 'subject'")
	}
	if !containsField(fields, "predicate") {
		t.Error("containsField should return true for 'predicate'")
	}
	if containsField(fields, "object") {
		t.Error("containsField should return false for 'object'")
	}
	if containsField([]string{}, "subject") {
		t.Error("containsField should return false for empty slice")
	}
}

func TestHasAllFields(t *testing.T) {
	tests := []struct {
		name     string
		triple   *graph.Triple
		expected bool
	}{
		{
			name:     "all fields",
			triple:   graph.NewTripleFromStrings("s", "p", "o"),
			expected: true,
		},
		{
			name:     "missing subject",
			triple:   &graph.Triple{Predicate: []byte("p"), Object: []byte("o")},
			expected: false,
		},
		{
			name:     "missing predicate",
			triple:   &graph.Triple{Subject: []byte("s"), Object: []byte("o")},
			expected: false,
		},
		{
			name:     "missing object",
			triple:   &graph.Triple{Subject: []byte("s"), Predicate: []byte("p")},
			expected: false,
		},
		{
			name:     "all nil",
			triple:   &graph.Triple{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAllFields(tt.triple)
			if result != tt.expected {
				t.Errorf("hasAllFields(%v) = %v, want %v", tt.triple, result, tt.expected)
			}
		})
	}
}
