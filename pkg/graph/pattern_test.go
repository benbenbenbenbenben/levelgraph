// Copyright (c) 2013-2024 Matteo Collina and LevelGraph Contributors
// Copyright (c) 2024 LevelGraph Go Contributors
//
// MIT License

package graph

import (
	"bytes"
	"testing"
)

func TestPatternValue_Wildcard(t *testing.T) {
	pv := Wildcard()

	if !pv.IsWildcard() {
		t.Error("expected IsWildcard() to be true")
	}
	if pv.IsExact() {
		t.Error("expected IsExact() to be false")
	}
	if pv.IsBinding() {
		t.Error("expected IsBinding() to be false")
	}
	if pv.Data() != nil {
		t.Error("expected Data() to be nil")
	}
	if pv.VariableName() != "" {
		t.Error("expected VariableName() to be empty")
	}
	if pv.ToInterface() != nil {
		t.Error("expected ToInterface() to return nil")
	}
}

func TestPatternValue_Exact(t *testing.T) {
	data := []byte("test-data")
	pv := Exact(data)

	if pv.IsWildcard() {
		t.Error("expected IsWildcard() to be false")
	}
	if !pv.IsExact() {
		t.Error("expected IsExact() to be true")
	}
	if pv.IsBinding() {
		t.Error("expected IsBinding() to be false")
	}
	if !bytes.Equal(pv.Data(), data) {
		t.Errorf("expected Data() to be %v, got %v", data, pv.Data())
	}
	if pv.VariableName() != "" {
		t.Error("expected VariableName() to be empty")
	}
	if result, ok := pv.ToInterface().([]byte); !ok || !bytes.Equal(result, data) {
		t.Error("expected ToInterface() to return the data bytes")
	}
}

func TestPatternValue_ExactString(t *testing.T) {
	s := "test-string"
	pv := ExactString(s)

	if pv.IsWildcard() {
		t.Error("expected IsWildcard() to be false")
	}
	if !pv.IsExact() {
		t.Error("expected IsExact() to be true")
	}
	if !bytes.Equal(pv.Data(), []byte(s)) {
		t.Errorf("expected Data() to be %v, got %v", []byte(s), pv.Data())
	}
}

func TestPatternValue_Binding(t *testing.T) {
	name := "x"
	pv := Binding(name)

	if pv.IsWildcard() {
		t.Error("expected IsWildcard() to be false")
	}
	if pv.IsExact() {
		t.Error("expected IsExact() to be false")
	}
	if !pv.IsBinding() {
		t.Error("expected IsBinding() to be true")
	}
	if pv.Data() != nil {
		t.Error("expected Data() to be nil")
	}
	if pv.VariableName() != name {
		t.Errorf("expected VariableName() to be %q, got %q", name, pv.VariableName())
	}
	if result, ok := pv.ToInterface().(*Variable); !ok || result.Name != name {
		t.Error("expected ToInterface() to return *Variable with correct name")
	}
}

func TestNewPattern(t *testing.T) {
	// Test with nil values (wildcards)
	p := NewPattern(nil, nil, nil)
	if !p.Subject.IsWildcard() || !p.Predicate.IsWildcard() || !p.Object.IsWildcard() {
		t.Error("nil values should create wildcards")
	}

	// Test with []byte values
	p = NewPattern([]byte("alice"), []byte("knows"), []byte("bob"))
	if !p.Subject.IsExact() || !bytes.Equal(p.Subject.Data(), []byte("alice")) {
		t.Error("[]byte subject should create exact match")
	}
	if !p.Predicate.IsExact() || !bytes.Equal(p.Predicate.Data(), []byte("knows")) {
		t.Error("[]byte predicate should create exact match")
	}
	if !p.Object.IsExact() || !bytes.Equal(p.Object.Data(), []byte("bob")) {
		t.Error("[]byte object should create exact match")
	}

	// Test with string values
	p = NewPattern("alice", "knows", "bob")
	if !p.Subject.IsExact() || !bytes.Equal(p.Subject.Data(), []byte("alice")) {
		t.Error("string subject should create exact match")
	}

	// Test with *Variable
	p = NewPattern(V("x"), []byte("knows"), V("y"))
	if !p.Subject.IsBinding() || p.Subject.VariableName() != "x" {
		t.Error("*Variable subject should create binding")
	}
	if !p.Object.IsBinding() || p.Object.VariableName() != "y" {
		t.Error("*Variable object should create binding")
	}

	// Test with PatternValue directly
	p = NewPattern(Binding("a"), Exact([]byte("rel")), Wildcard())
	if !p.Subject.IsBinding() || p.Subject.VariableName() != "a" {
		t.Error("PatternValue Binding should be preserved")
	}
	if !p.Predicate.IsExact() || !bytes.Equal(p.Predicate.Data(), []byte("rel")) {
		t.Error("PatternValue Exact should be preserved")
	}
	if !p.Object.IsWildcard() {
		t.Error("PatternValue Wildcard should be preserved")
	}

	// Test with empty string (should be wildcard)
	p = NewPattern("", nil, nil)
	if !p.Subject.IsWildcard() {
		t.Error("empty string should create wildcard")
	}

	// Test with empty []byte (should be wildcard)
	p = NewPattern([]byte{}, nil, nil)
	if !p.Subject.IsWildcard() {
		t.Error("empty []byte should create wildcard")
	}

	// Test with boolean
	p = NewPattern(true, false, nil)
	if !p.Subject.IsExact() || string(p.Subject.Data()) != "true" {
		t.Error("boolean true should create exact match 'true'")
	}
	if !p.Predicate.IsExact() || string(p.Predicate.Data()) != "false" {
		t.Error("boolean false should create exact match 'false'")
	}
}

func TestPattern_GetConcreteValue(t *testing.T) {
	p := NewPattern([]byte("alice"), V("x"), nil)

	if !bytes.Equal(p.GetConcreteValue("subject"), []byte("alice")) {
		t.Error("GetConcreteValue should return subject value")
	}
	if p.GetConcreteValue("predicate") != nil {
		t.Error("GetConcreteValue should return nil for variable")
	}
	if p.GetConcreteValue("object") != nil {
		t.Error("GetConcreteValue should return nil for wildcard")
	}
	if p.GetConcreteValue("invalid") != nil {
		t.Error("GetConcreteValue should return nil for invalid field")
	}
}

func TestPattern_GetVariable(t *testing.T) {
	p := NewPattern(V("x"), []byte("knows"), V("y"))

	if v := p.GetVariable("subject"); v == nil || v.Name != "x" {
		t.Error("GetVariable should return subject variable")
	}
	if p.GetVariable("predicate") != nil {
		t.Error("GetVariable should return nil for concrete value")
	}
	if v := p.GetVariable("object"); v == nil || v.Name != "y" {
		t.Error("GetVariable should return object variable")
	}
	if p.GetVariable("invalid") != nil {
		t.Error("GetVariable should return nil for invalid field")
	}
}

func TestPattern_HasVariable(t *testing.T) {
	p := NewPattern([]byte("alice"), []byte("knows"), []byte("bob"))
	if p.HasVariable() {
		t.Error("pattern with no variables should return false")
	}

	p = NewPattern(V("x"), []byte("knows"), []byte("bob"))
	if !p.HasVariable() {
		t.Error("pattern with subject variable should return true")
	}

	p = NewPattern([]byte("alice"), V("y"), []byte("bob"))
	if !p.HasVariable() {
		t.Error("pattern with predicate variable should return true")
	}

	p = NewPattern([]byte("alice"), []byte("knows"), V("z"))
	if !p.HasVariable() {
		t.Error("pattern with object variable should return true")
	}
}

func TestPattern_ConcreteFields(t *testing.T) {
	p := NewPattern([]byte("alice"), V("x"), []byte("bob"))
	fields := p.ConcreteFields()

	if len(fields) != 2 {
		t.Errorf("expected 2 concrete fields, got %d", len(fields))
	}
	if fields[0] != "subject" || fields[1] != "object" {
		t.Errorf("expected [subject, object], got %v", fields)
	}
}

func TestPattern_VariableFields(t *testing.T) {
	p := NewPattern(V("x"), []byte("knows"), V("y"))
	vars := p.VariableFields()

	if len(vars) != 2 {
		t.Errorf("expected 2 variables, got %d", len(vars))
	}
	if vars["subject"] == nil || vars["subject"].Name != "x" {
		t.Error("expected subject variable 'x'")
	}
	if vars["object"] == nil || vars["object"].Name != "y" {
		t.Error("expected object variable 'y'")
	}
}

func TestPattern_ToTriple(t *testing.T) {
	// Full concrete pattern should convert to triple
	p := NewPattern([]byte("alice"), []byte("knows"), []byte("bob"))
	triple := p.ToTriple()
	if triple == nil {
		t.Fatal("expected triple, got nil")
	}
	if !bytes.Equal(triple.Subject, []byte("alice")) {
		t.Error("subject mismatch")
	}
	if !bytes.Equal(triple.Predicate, []byte("knows")) {
		t.Error("predicate mismatch")
	}
	if !bytes.Equal(triple.Object, []byte("bob")) {
		t.Error("object mismatch")
	}

	// Pattern with wildcard should return nil
	p = NewPattern(nil, []byte("knows"), []byte("bob"))
	if p.ToTriple() != nil {
		t.Error("pattern with wildcard should return nil triple")
	}

	// Pattern with variable should return nil
	p = NewPattern(V("x"), []byte("knows"), []byte("bob"))
	if p.ToTriple() != nil {
		t.Error("pattern with variable should return nil triple")
	}
}

func TestPattern_Matches(t *testing.T) {
	triple := &Triple{
		Subject:   []byte("alice"),
		Predicate: []byte("knows"),
		Object:    []byte("bob"),
	}

	// Wildcard matches everything
	p := NewPattern(nil, nil, nil)
	if !p.Matches(triple) {
		t.Error("wildcard pattern should match")
	}

	// Exact match
	p = NewPattern([]byte("alice"), []byte("knows"), []byte("bob"))
	if !p.Matches(triple) {
		t.Error("exact pattern should match")
	}

	// Partial match
	p = NewPattern([]byte("alice"), nil, nil)
	if !p.Matches(triple) {
		t.Error("partial pattern should match")
	}

	// Non-match on subject
	p = NewPattern([]byte("eve"), nil, nil)
	if p.Matches(triple) {
		t.Error("non-matching subject should not match")
	}

	// Non-match on predicate
	p = NewPattern(nil, []byte("hates"), nil)
	if p.Matches(triple) {
		t.Error("non-matching predicate should not match")
	}

	// Non-match on object
	p = NewPattern(nil, nil, []byte("eve"))
	if p.Matches(triple) {
		t.Error("non-matching object should not match")
	}

	// Variable patterns should match (variables act as wildcards in matching)
	p = NewPattern(V("x"), []byte("knows"), V("y"))
	if !p.Matches(triple) {
		t.Error("pattern with variables should match")
	}
}

func TestPattern_UpdateWithSolution(t *testing.T) {
	p := NewPattern(V("x"), []byte("knows"), V("y"))
	solution := Solution{
		"x": []byte("alice"),
		"y": []byte("bob"),
	}

	updated := p.UpdateWithSolution(solution)

	if !updated.Subject.IsExact() || !bytes.Equal(updated.Subject.Data(), []byte("alice")) {
		t.Error("subject should be updated from solution")
	}
	if !updated.Predicate.IsExact() || !bytes.Equal(updated.Predicate.Data(), []byte("knows")) {
		t.Error("predicate should remain unchanged")
	}
	if !updated.Object.IsExact() || !bytes.Equal(updated.Object.Data(), []byte("bob")) {
		t.Error("object should be updated from solution")
	}

	// Test with partial solution
	p = NewPattern(V("x"), V("y"), V("z"))
	solution = Solution{"x": []byte("alice")}
	updated = p.UpdateWithSolution(solution)

	if !updated.Subject.IsExact() {
		t.Error("subject should be updated")
	}
	if !updated.Predicate.IsBinding() {
		t.Error("predicate should remain a binding")
	}
	if !updated.Object.IsBinding() {
		t.Error("object should remain a binding")
	}
}

func TestPattern_BindTriple(t *testing.T) {
	triple := &Triple{
		Subject:   []byte("alice"),
		Predicate: []byte("knows"),
		Object:    []byte("bob"),
	}

	// Bind to empty solution
	p := NewPattern(V("x"), []byte("knows"), V("y"))
	result := p.BindTriple(nil, triple)
	if result == nil {
		t.Fatal("expected solution, got nil")
	}
	if !bytes.Equal(result["x"], []byte("alice")) {
		t.Error("x should be bound to alice")
	}
	if !bytes.Equal(result["y"], []byte("bob")) {
		t.Error("y should be bound to bob")
	}

	// Bind with existing solution that matches
	solution := Solution{"x": []byte("alice")}
	result = p.BindTriple(solution, triple)
	if result == nil {
		t.Fatal("expected solution, got nil")
	}

	// Bind with existing solution that conflicts
	solution = Solution{"x": []byte("eve")}
	result = p.BindTriple(solution, triple)
	if result != nil {
		t.Error("conflicting binding should return nil")
	}

	// Concrete pattern that doesn't match
	p = NewPattern([]byte("eve"), []byte("knows"), []byte("bob"))
	result = p.BindTriple(nil, triple)
	if result != nil {
		t.Error("non-matching pattern should return nil")
	}

	// Concrete pattern that matches
	p = NewPattern([]byte("alice"), []byte("knows"), []byte("bob"))
	result = p.BindTriple(nil, triple)
	if result == nil {
		t.Fatal("matching concrete pattern should return solution")
	}
}

func TestPattern_PreservesOptions(t *testing.T) {
	filter := func(t *Triple) bool { return true }
	p := &Pattern{
		Subject:   Exact([]byte("alice")),
		Predicate: Wildcard(),
		Object:    Binding("x"),
		Filter:    filter,
		Limit:     10,
		Offset:    5,
		Reverse:   true,
	}

	solution := Solution{"x": []byte("bob")}
	updated := p.UpdateWithSolution(solution)

	if updated.Limit != 10 {
		t.Error("Limit should be preserved")
	}
	if updated.Offset != 5 {
		t.Error("Offset should be preserved")
	}
	if !updated.Reverse {
		t.Error("Reverse should be preserved")
	}
	if updated.Filter == nil {
		t.Error("Filter should be preserved")
	}
}
