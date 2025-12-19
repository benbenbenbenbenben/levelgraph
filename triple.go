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
	"bytes"
	"encoding/base64"
	"encoding/json"
)

// Triple represents a subject-predicate-object triple in the graph database.
// All values are stored as []byte for portability and binary data support.
type Triple struct {
	Subject   []byte `json:"subject"`
	Predicate []byte `json:"predicate"`
	Object    []byte `json:"object"`
}

// NewTriple creates a new Triple from byte slices.
func NewTriple(subject, predicate, object []byte) *Triple {
	return &Triple{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
	}
}

// NewTripleFromStrings creates a new Triple from strings (convenience function).
func NewTripleFromStrings(subject, predicate, object string) *Triple {
	return &Triple{
		Subject:   []byte(subject),
		Predicate: []byte(predicate),
		Object:    []byte(object),
	}
}

// Clone creates a deep copy of the triple.
func (t *Triple) Clone() *Triple {
	return &Triple{
		Subject:   bytes.Clone(t.Subject),
		Predicate: bytes.Clone(t.Predicate),
		Object:    bytes.Clone(t.Object),
	}
}

// Equal returns true if two triples have identical subject, predicate, and object.
func (t *Triple) Equal(other *Triple) bool {
	if other == nil {
		return false
	}
	return bytes.Equal(t.Subject, other.Subject) &&
		bytes.Equal(t.Predicate, other.Predicate) &&
		bytes.Equal(t.Object, other.Object)
}

// String returns a human-readable representation of the triple.
func (t *Triple) String() string {
	return string(t.Subject) + " " + string(t.Predicate) + " " + string(t.Object)
}

// tripleJSON is used for JSON marshaling/unmarshaling with base64 support
type tripleJSON struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

// MarshalJSON implements json.Marshaler for Triple.
// Uses base64 encoding for binary data to preserve all byte values.
func (t *Triple) MarshalJSON() ([]byte, error) {
	return json.Marshal(tripleJSON{
		Subject:   base64.StdEncoding.EncodeToString(t.Subject),
		Predicate: base64.StdEncoding.EncodeToString(t.Predicate),
		Object:    base64.StdEncoding.EncodeToString(t.Object),
	})
}

// UnmarshalJSON implements json.Unmarshaler for Triple.
func (t *Triple) UnmarshalJSON(data []byte) error {
	var tj tripleJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return err
	}

	var err error
	t.Subject, err = base64.StdEncoding.DecodeString(tj.Subject)
	if err != nil {
		return err
	}
	t.Predicate, err = base64.StdEncoding.DecodeString(tj.Predicate)
	if err != nil {
		return err
	}
	t.Object, err = base64.StdEncoding.DecodeString(tj.Object)
	if err != nil {
		return err
	}
	return nil
}

// Get returns the value at the specified position (subject, predicate, or object).
func (t *Triple) Get(field string) []byte {
	switch field {
	case "subject":
		return t.Subject
	case "predicate":
		return t.Predicate
	case "object":
		return t.Object
	default:
		return nil
	}
}

// Set sets the value at the specified position.
func (t *Triple) Set(field string, value []byte) {
	switch field {
	case "subject":
		t.Subject = value
	case "predicate":
		t.Predicate = value
	case "object":
		t.Object = value
	}
}
