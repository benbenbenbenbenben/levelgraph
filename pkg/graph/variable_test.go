// Copyright (c) 2013-2024 Matteo Collina and LevelGraph Contributors
// Copyright (c) 2024 LevelGraph Go Contributors
//
// MIT License

package graph

import (
	"bytes"
	"testing"
)

func TestV(t *testing.T) {
	v := V("test")
	if v == nil {
		t.Fatal("V() should return non-nil")
	}
	if v.Name != "test" {
		t.Errorf("expected Name %q, got %q", "test", v.Name)
	}
}

func TestVariable_Bind(t *testing.T) {
	v := V("x")
	value := []byte("alice")

	// Bind to empty solution
	result := v.Bind(nil, value)
	if result == nil {
		t.Fatal("expected solution, got nil")
	}
	if !bytes.Equal(result["x"], value) {
		t.Error("variable should be bound to value")
	}

	// Bind to existing solution without conflict
	existing := Solution{"y": []byte("bob")}
	result = v.Bind(existing, value)
	if result == nil {
		t.Fatal("expected solution, got nil")
	}
	if !bytes.Equal(result["x"], value) {
		t.Error("x should be bound")
	}
	if !bytes.Equal(result["y"], []byte("bob")) {
		t.Error("y should be preserved")
	}

	// Bind to same value (should succeed)
	existing = Solution{"x": value}
	result = v.Bind(existing, value)
	if result == nil {
		t.Fatal("binding to same value should succeed")
	}

	// Bind to different value (should fail)
	existing = Solution{"x": []byte("eve")}
	result = v.Bind(existing, value)
	if result != nil {
		t.Error("binding to different value should fail")
	}
}

func TestVariable_IsBound(t *testing.T) {
	v := V("x")

	// Not bound
	solution := Solution{}
	if v.IsBound(solution) {
		t.Error("should not be bound in empty solution")
	}

	// Bound
	solution = Solution{"x": []byte("alice")}
	if !v.IsBound(solution) {
		t.Error("should be bound")
	}
}

func TestVariable_IsBindable(t *testing.T) {
	v := V("x")
	value := []byte("alice")

	// Unbound - always bindable
	if !v.IsBindable(Solution{}, value) {
		t.Error("unbound variable should be bindable")
	}

	// Bound to same value - bindable
	solution := Solution{"x": value}
	if !v.IsBindable(solution, value) {
		t.Error("variable bound to same value should be bindable")
	}

	// Bound to different value - not bindable
	solution = Solution{"x": []byte("bob")}
	if v.IsBindable(solution, value) {
		t.Error("variable bound to different value should not be bindable")
	}
}

func TestVariable_GetValue(t *testing.T) {
	v := V("x")

	// Unbound
	if v.GetValue(Solution{}) != nil {
		t.Error("unbound variable should return nil")
	}

	// Bound
	value := []byte("alice")
	solution := Solution{"x": value}
	if !bytes.Equal(v.GetValue(solution), value) {
		t.Error("bound variable should return value")
	}
}

func TestSolution_Clone(t *testing.T) {
	// Clone nil
	var s Solution = nil
	if s.Clone() != nil {
		t.Error("cloning nil should return nil")
	}

	// Clone empty
	s = Solution{}
	clone := s.Clone()
	if clone == nil || len(clone) != 0 {
		t.Error("cloning empty should return empty")
	}

	// Clone with data
	s = Solution{
		"x": []byte("alice"),
		"y": []byte("bob"),
	}
	clone = s.Clone()
	if len(clone) != 2 {
		t.Error("clone should have same length")
	}
	if !bytes.Equal(clone["x"], []byte("alice")) || !bytes.Equal(clone["y"], []byte("bob")) {
		t.Error("clone should have same values")
	}

	// Verify deep copy
	clone["x"][0] = 'X'
	if s["x"][0] == 'X' {
		t.Error("clone should be a deep copy")
	}
}

func TestSolution_Equal(t *testing.T) {
	s1 := Solution{"x": []byte("alice"), "y": []byte("bob")}
	s2 := Solution{"x": []byte("alice"), "y": []byte("bob")}
	s3 := Solution{"x": []byte("alice")}
	s4 := Solution{"x": []byte("alice"), "y": []byte("eve")}

	if !s1.Equal(s2) {
		t.Error("identical solutions should be equal")
	}
	if s1.Equal(s3) {
		t.Error("solutions with different lengths should not be equal")
	}
	if s1.Equal(s4) {
		t.Error("solutions with different values should not be equal")
	}
}

func TestIsVariable(t *testing.T) {
	// *Variable
	if !IsVariable(V("x")) {
		t.Error("*Variable should return true")
	}

	// PatternValue binding
	if !IsVariable(Binding("x")) {
		t.Error("PatternValue binding should return true")
	}

	// PatternValue exact
	if IsVariable(Exact([]byte("test"))) {
		t.Error("PatternValue exact should return false")
	}

	// PatternValue wildcard
	if IsVariable(Wildcard()) {
		t.Error("PatternValue wildcard should return false")
	}

	// Other types
	if IsVariable([]byte("test")) {
		t.Error("[]byte should return false")
	}
	if IsVariable("test") {
		t.Error("string should return false")
	}
	if IsVariable(nil) {
		t.Error("nil should return false")
	}
}

func TestAsVariable(t *testing.T) {
	// *Variable
	v, ok := AsVariable(V("x"))
	if !ok || v == nil || v.Name != "x" {
		t.Error("*Variable should convert successfully")
	}

	// PatternValue binding
	v, ok = AsVariable(Binding("y"))
	if !ok || v == nil || v.Name != "y" {
		t.Error("PatternValue binding should convert successfully")
	}

	// PatternValue exact
	v, ok = AsVariable(Exact([]byte("test")))
	if ok || v != nil {
		t.Error("PatternValue exact should not convert")
	}

	// PatternValue wildcard
	v, ok = AsVariable(Wildcard())
	if ok || v != nil {
		t.Error("PatternValue wildcard should not convert")
	}

	// Other types
	v, ok = AsVariable([]byte("test"))
	if ok || v != nil {
		t.Error("[]byte should not convert")
	}

	v, ok = AsVariable(nil)
	if ok || v != nil {
		t.Error("nil should not convert")
	}
}

func TestVarAlias(t *testing.T) {
	// Var is an alias for Variable
	v := Var{Name: "test"}
	if v.Name != "test" {
		t.Error("Var alias should work")
	}
}

func TestVariable_BindInPlace(t *testing.T) {
	v := V("x")
	value := []byte("alice")

	// Bind to empty solution
	solution := Solution{}
	if !v.BindInPlace(solution, value) {
		t.Error("BindInPlace should succeed on empty solution")
	}
	if !bytes.Equal(solution["x"], value) {
		t.Error("variable should be bound to value")
	}

	// Bind to same value (should succeed)
	if !v.BindInPlace(solution, value) {
		t.Error("BindInPlace to same value should succeed")
	}

	// Bind to different value (should fail)
	if v.BindInPlace(solution, []byte("bob")) {
		t.Error("BindInPlace to different value should fail")
	}

	// Original value should be unchanged
	if !bytes.Equal(solution["x"], value) {
		t.Error("original value should be unchanged after failed bind")
	}
}

func TestSolution_ShallowClone(t *testing.T) {
	// Clone nil
	var s Solution = nil
	if s.ShallowClone() != nil {
		t.Error("shallow cloning nil should return nil")
	}

	// Clone empty
	s = Solution{}
	clone := s.ShallowClone()
	if clone == nil || len(clone) != 0 {
		t.Error("shallow cloning empty should return empty")
	}

	// Clone with data
	s = Solution{
		"x": []byte("alice"),
		"y": []byte("bob"),
	}
	clone = s.ShallowClone()
	if len(clone) != 2 {
		t.Error("shallow clone should have same length")
	}
	if !bytes.Equal(clone["x"], []byte("alice")) || !bytes.Equal(clone["y"], []byte("bob")) {
		t.Error("shallow clone should have same values")
	}

	// Verify shallow copy (shared references)
	// Modifying the byte slice in clone should affect original
	clone["x"][0] = 'X'
	if s["x"][0] != 'X' {
		t.Error("shallow clone should share byte slice references")
	}

	// But adding new keys doesn't affect original
	clone["z"] = []byte("new")
	if _, exists := s["z"]; exists {
		t.Error("adding to shallow clone shouldn't affect original map")
	}
}
