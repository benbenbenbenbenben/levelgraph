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
	"sort"

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
)

// IndexName represents the name of a hexastore index.
type IndexName string

const (
	IndexSPO IndexName = "spo"
	IndexSOP IndexName = "sop"
	IndexPOS IndexName = "pos"
	IndexPSO IndexName = "pso"
	IndexOPS IndexName = "ops"
	IndexOSP IndexName = "osp"
)

// IndexDef defines the order of fields for each index.
// Each index stores the same triple data but with different key orderings
// to enable efficient lookups based on which components are specified.
var IndexDefs = map[IndexName][]string{
	IndexSPO: {"subject", "predicate", "object"},
	IndexSOP: {"subject", "object", "predicate"},
	IndexPOS: {"predicate", "object", "subject"},
	IndexPSO: {"predicate", "subject", "object"},
	IndexOPS: {"object", "predicate", "subject"},
	IndexOSP: {"object", "subject", "predicate"},
}

// AllIndexes returns all index names in a consistent order.
var AllIndexes = []IndexName{IndexSPO, IndexSOP, IndexPOS, IndexPSO, IndexOPS, IndexOSP}

// KeySeparator used between key components.
var KeySeparator = []byte("::")

// UpperBound is the upper bound character for range queries.
// Using 0xFF bytes as the upper bound since they are the highest byte values
// and will be greater than any valid UTF-8 sequence.
var upperBound = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

// Escape escapes special characters in a value for use in keys.
// Escapes backslash (\) and colon (:) characters.
func Escape(value []byte) []byte {
	if value == nil {
		return nil
	}

	// Count how many characters need escaping
	escapeCount := 0
	for _, b := range value {
		if b == '\\' || b == ':' {
			escapeCount++
		}
	}

	if escapeCount == 0 {
		return value
	}

	// Create new buffer with space for escapes
	result := make([]byte, 0, len(value)+escapeCount)
	for _, b := range value {
		if b == '\\' || b == ':' {
			result = append(result, '\\')
		}
		result = append(result, b)
	}
	return result
}

// Unescape reverses the escaping done by Escape.
func Unescape(value []byte) []byte {
	if value == nil {
		return nil
	}

	// Check if there are any escape sequences
	hasEscapes := false
	for i := 0; i < len(value)-1; i++ {
		if value[i] == '\\' {
			hasEscapes = true
			break
		}
	}

	if !hasEscapes {
		return value
	}

	result := make([]byte, 0, len(value))
	i := 0
	for i < len(value) {
		if i < len(value)-1 && value[i] == '\\' && (value[i+1] == '\\' || value[i+1] == ':') {
			result = append(result, value[i+1])
			i += 2
		} else {
			result = append(result, value[i])
			i++
		}
	}
	return result
}

// GenKey generates a key for a single index from a triple.
// The key format is: indexName::value1::value2::value3
func GenKey(index IndexName, triple *graph.Triple) []byte {
	def := IndexDefs[index]
	var buf bytes.Buffer

	buf.WriteString(string(index))

	for _, field := range def {
		value := triple.Get(field)
		if value == nil {
			break
		}
		buf.Write(KeySeparator)
		buf.Write(Escape(value))
	}

	// Add trailing separator if not all fields present
	if !hasAllFields(triple) {
		buf.Write(KeySeparator)
	}

	return buf.Bytes()
}

// GenKeyFromPattern generates a key for a single index from a pattern.
// Unlike GenKey, this handles partial patterns where some fields may be nil or variables.
func GenKeyFromPattern(index IndexName, pattern *graph.Pattern) []byte {
	def := IndexDefs[index]
	var buf bytes.Buffer

	buf.WriteString(string(index))

	concreteCount := 0
	for _, field := range def {
		value := pattern.GetConcreteValue(field)
		if value == nil {
			break
		}
		buf.Write(KeySeparator)
		buf.Write(Escape(value))
		concreteCount++
	}

	// Only add trailing separator if we don't have all 3 fields
	// (this makes range queries work correctly)
	if concreteCount < 3 {
		buf.Write(KeySeparator)
	}

	return buf.Bytes()
}

// GenKeyWithUpperBound generates a key with upper bound for range queries.
func GenKeyWithUpperBound(index IndexName, pattern *graph.Pattern) []byte {
	key := GenKeyFromPattern(index, pattern)

	// Check if we have all three components
	concreteCount := 0
	for _, field := range IndexDefs[index] {
		if pattern.GetConcreteValue(field) != nil {
			concreteCount++
		} else {
			break
		}
	}

	// If we have all 3 fields, we still need an upper bound for range query to work.
	// LevelDB range is [start, limit), so if start == limit, nothing is returned.
	// Add a byte to make the range inclusive of the exact key.
	if concreteCount == 3 {
		return append(key, 0xFF)
	}

	return append(key, upperBound...)
}

// GenKeys generates keys for all six indexes from a triple.
func GenKeys(triple *graph.Triple) [][]byte {
	keys := make([][]byte, len(AllIndexes))
	for i, index := range AllIndexes {
		keys[i] = GenKey(index, triple)
	}
	return keys
}

// hasAllFields returns true if the triple has all three fields set.
func hasAllFields(triple *graph.Triple) bool {
	return triple.Subject != nil && triple.Predicate != nil && triple.Object != nil
}

// PossibleIndexes returns indexes that can efficiently query the given field types.
// The fields slice contains the fields that have concrete (non-variable) values.
// An index is efficient if the specified fields form a prefix of the index definition.
func PossibleIndexes(fields []string) []IndexName {
	var result []IndexName

	for _, index := range AllIndexes {
		def := IndexDefs[index]

		// Check if the fields form a prefix of this index's definition
		// i.e., all specified fields must appear in order at the start
		isPrefix := true
		for i, field := range fields {
			if i >= len(def) || !containsField(fields, def[i]) {
				isPrefix = false
				break
			}
			// Also verify all fields are accounted for in the prefix
			found := false
			for j := 0; j <= i && j < len(def); j++ {
				if def[j] == field {
					found = true
					break
				}
			}
			if !found {
				isPrefix = false
				break
			}
		}

		// More strict check: ensure the first len(fields) of def are exactly our fields
		if len(fields) > 0 && isPrefix {
			isPrefix = true
			for i := 0; i < len(fields); i++ {
				if !containsField(fields, def[i]) {
					isPrefix = false
					break
				}
			}
		}

		if isPrefix {
			result = append(result, index)
		}
	}

	// Sort for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}

// containsField checks if a field is in the list of fields
func containsField(fields []string, field string) bool {
	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
}

// FindIndex finds the best index for querying the given pattern.
// If preferredIndex is provided and valid, it will be used.
func FindIndex(fields []string, preferredIndex IndexName) IndexName {
	possible := PossibleIndexes(fields)

	if len(possible) == 0 {
		// Default to SPO if no specific index matches
		return IndexSPO
	}

	// Check if preferred index is among possibilities
	for _, idx := range possible {
		if idx == preferredIndex {
			return preferredIndex
		}
	}

	return possible[0]
}

// ParseKey parses a key back into its components.
// Returns the index name and the field values.
func ParseKey(key []byte) (IndexName, [][]byte) {
	parts := bytes.Split(key, KeySeparator)
	if len(parts) == 0 {
		return "", nil
	}

	indexName := IndexName(parts[0])
	values := make([][]byte, 0, 3)

	for i := 1; i < len(parts) && i <= 3; i++ {
		if len(parts[i]) > 0 {
			values = append(values, Unescape(parts[i]))
		}
	}

	return indexName, values
}
