// Copyright (c) 2013-2024 Matteo Collina and LevelGraph Contributors
// Copyright (c) 2024 LevelGraph Go Contributors
//
// MIT License

package graph

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewTriple(t *testing.T) {
	subject := []byte("alice")
	predicate := []byte("knows")
	object := []byte("bob")

	triple := NewTriple(subject, predicate, object)

	if triple == nil {
		t.Fatal("NewTriple should return non-nil")
	}
	if !bytes.Equal(triple.Subject, subject) {
		t.Errorf("Subject mismatch: got %v, want %v", triple.Subject, subject)
	}
	if !bytes.Equal(triple.Predicate, predicate) {
		t.Errorf("Predicate mismatch: got %v, want %v", triple.Predicate, predicate)
	}
	if !bytes.Equal(triple.Object, object) {
		t.Errorf("Object mismatch: got %v, want %v", triple.Object, object)
	}
}

func TestNewTripleFromStrings(t *testing.T) {
	triple := NewTripleFromStrings("alice", "knows", "bob")

	if triple == nil {
		t.Fatal("NewTripleFromStrings should return non-nil")
	}
	if string(triple.Subject) != "alice" {
		t.Errorf("Subject mismatch: got %s, want alice", triple.Subject)
	}
	if string(triple.Predicate) != "knows" {
		t.Errorf("Predicate mismatch: got %s, want knows", triple.Predicate)
	}
	if string(triple.Object) != "bob" {
		t.Errorf("Object mismatch: got %s, want bob", triple.Object)
	}
}

func TestTriple_Clone(t *testing.T) {
	original := NewTripleFromStrings("alice", "knows", "bob")
	clone := original.Clone()

	// Verify values are equal
	if !bytes.Equal(clone.Subject, original.Subject) {
		t.Error("Clone subject should equal original")
	}
	if !bytes.Equal(clone.Predicate, original.Predicate) {
		t.Error("Clone predicate should equal original")
	}
	if !bytes.Equal(clone.Object, original.Object) {
		t.Error("Clone object should equal original")
	}

	// Verify it's a deep copy (modifying clone doesn't affect original)
	clone.Subject[0] = 'X'
	if original.Subject[0] == 'X' {
		t.Error("Clone should be a deep copy")
	}
}

func TestTriple_Equal(t *testing.T) {
	t1 := NewTripleFromStrings("alice", "knows", "bob")
	t2 := NewTripleFromStrings("alice", "knows", "bob")
	t3 := NewTripleFromStrings("alice", "knows", "eve")
	t4 := NewTripleFromStrings("alice", "likes", "bob")
	t5 := NewTripleFromStrings("eve", "knows", "bob")

	if !t1.Equal(t2) {
		t.Error("Identical triples should be equal")
	}
	if t1.Equal(t3) {
		t.Error("Triples with different objects should not be equal")
	}
	if t1.Equal(t4) {
		t.Error("Triples with different predicates should not be equal")
	}
	if t1.Equal(t5) {
		t.Error("Triples with different subjects should not be equal")
	}
	if t1.Equal(nil) {
		t.Error("Triple should not equal nil")
	}
}

func TestTriple_String(t *testing.T) {
	triple := NewTripleFromStrings("alice", "knows", "bob")
	expected := "alice knows bob"

	if triple.String() != expected {
		t.Errorf("String() = %q, want %q", triple.String(), expected)
	}
}

func TestTriple_MarshalJSON(t *testing.T) {
	triple := NewTripleFromStrings("alice", "knows", "bob")

	data, err := json.Marshal(triple)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Should be valid JSON
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Result should be valid JSON: %v", err)
	}

	// Should have subject, predicate, object fields (base64 encoded)
	if _, ok := result["subject"]; !ok {
		t.Error("JSON should have subject field")
	}
	if _, ok := result["predicate"]; !ok {
		t.Error("JSON should have predicate field")
	}
	if _, ok := result["object"]; !ok {
		t.Error("JSON should have object field")
	}
}

func TestTriple_UnmarshalJSON(t *testing.T) {
	original := NewTripleFromStrings("alice", "knows", "bob")

	// Marshal then unmarshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var restored Triple
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if !original.Equal(&restored) {
		t.Errorf("Restored triple doesn't match original: got %v, want %v", &restored, original)
	}
}

func TestTriple_JSON_BinaryData(t *testing.T) {
	// Test with binary data (non-UTF8)
	original := &Triple{
		Subject:   []byte{0x00, 0xFF, 0x80},
		Predicate: []byte{0x01, 0x02, 0x03},
		Object:    []byte{0xFE, 0xFD, 0xFC},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("MarshalJSON with binary data failed: %v", err)
	}

	var restored Triple
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("UnmarshalJSON with binary data failed: %v", err)
	}

	if !original.Equal(&restored) {
		t.Errorf("Binary data round-trip failed: got %v, want %v", &restored, original)
	}
}

func TestTriple_UnmarshalJSON_Invalid(t *testing.T) {
	var triple Triple

	// Invalid JSON
	if err := triple.UnmarshalJSON([]byte("not json")); err == nil {
		t.Error("Should fail on invalid JSON")
	}

	// Invalid base64 in subject
	if err := triple.UnmarshalJSON([]byte(`{"subject":"!!!","predicate":"","object":""}`)); err == nil {
		t.Error("Should fail on invalid base64 in subject")
	}

	// Invalid base64 in predicate
	if err := triple.UnmarshalJSON([]byte(`{"subject":"YWxpY2U=","predicate":"!!!","object":""}`)); err == nil {
		t.Error("Should fail on invalid base64 in predicate")
	}

	// Invalid base64 in object
	if err := triple.UnmarshalJSON([]byte(`{"subject":"YWxpY2U=","predicate":"a25vd3M=","object":"!!!"}`)); err == nil {
		t.Error("Should fail on invalid base64 in object")
	}
}

func TestTriple_MarshalBinary(t *testing.T) {
	original := NewTripleFromStrings("alice", "knows", "bob")

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("MarshalBinary should return non-empty data")
	}
}

func TestTriple_UnmarshalBinary(t *testing.T) {
	original := NewTripleFromStrings("alice", "knows", "bob")

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	var restored Triple
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if !original.Equal(&restored) {
		t.Errorf("Restored triple doesn't match original: got %v, want %v", &restored, original)
	}
}

func TestTriple_Binary_LargeData(t *testing.T) {
	// Test with larger data to exercise varint encoding
	original := &Triple{
		Subject:   bytes.Repeat([]byte("a"), 1000),
		Predicate: bytes.Repeat([]byte("b"), 500),
		Object:    bytes.Repeat([]byte("c"), 2000),
	}

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary with large data failed: %v", err)
	}

	var restored Triple
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary with large data failed: %v", err)
	}

	if !original.Equal(&restored) {
		t.Error("Large data round-trip failed")
	}
}

func TestTriple_UnmarshalBinary_Invalid(t *testing.T) {
	var triple Triple

	// Empty data
	if err := triple.UnmarshalBinary([]byte{}); err == nil {
		t.Error("Should fail on empty data")
	}

	// Truncated data (only subject length, no data)
	if err := triple.UnmarshalBinary([]byte{0x05}); err == nil {
		t.Error("Should fail on truncated data")
	}

	// Truncated after subject
	original := NewTripleFromStrings("alice", "knows", "bob")
	data, _ := original.MarshalBinary()
	if err := triple.UnmarshalBinary(data[:5]); err == nil {
		t.Error("Should fail on truncated after subject")
	}
}

func TestTriple_Get(t *testing.T) {
	triple := NewTripleFromStrings("alice", "knows", "bob")

	if !bytes.Equal(triple.Get("subject"), []byte("alice")) {
		t.Error("Get('subject') should return subject")
	}
	if !bytes.Equal(triple.Get("predicate"), []byte("knows")) {
		t.Error("Get('predicate') should return predicate")
	}
	if !bytes.Equal(triple.Get("object"), []byte("bob")) {
		t.Error("Get('object') should return object")
	}
	if triple.Get("invalid") != nil {
		t.Error("Get with invalid field should return nil")
	}
}

func TestTriple_Set(t *testing.T) {
	triple := &Triple{}

	triple.Set("subject", []byte("alice"))
	if !bytes.Equal(triple.Subject, []byte("alice")) {
		t.Error("Set('subject') should set subject")
	}

	triple.Set("predicate", []byte("knows"))
	if !bytes.Equal(triple.Predicate, []byte("knows")) {
		t.Error("Set('predicate') should set predicate")
	}

	triple.Set("object", []byte("bob"))
	if !bytes.Equal(triple.Object, []byte("bob")) {
		t.Error("Set('object') should set object")
	}

	// Invalid field should be no-op
	triple.Set("invalid", []byte("value"))
	// Just verify it doesn't panic
}

func TestTriple_BinaryJSONRoundTrip(t *testing.T) {
	// Verify that binary and JSON produce equivalent results
	original := NewTripleFromStrings("alice", "knows", "bob")

	// Binary round-trip
	binData, _ := original.MarshalBinary()
	var binRestored Triple
	binRestored.UnmarshalBinary(binData)

	// JSON round-trip
	jsonData, _ := json.Marshal(original)
	var jsonRestored Triple
	json.Unmarshal(jsonData, &jsonRestored)

	if !binRestored.Equal(&jsonRestored) {
		t.Error("Binary and JSON round-trips should produce identical results")
	}
}
