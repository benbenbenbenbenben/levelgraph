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

package levelgraph

import (
	"bytes"
	"testing"

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
)

func TestPatternValue_Wildcard(t *testing.T) {
	t.Parallel()
	pv := graph.Wildcard()
	if !pv.IsWildcard() {
		t.Error("expected IsWildcard to be true")
	}
	if pv.IsExact() {
		t.Error("expected IsExact to be false")
	}
	if pv.IsBinding() {
		t.Error("expected IsBinding to be false")
	}
	if pv.ToInterface() != nil {
		t.Error("expected ToInterface to return nil")
	}
}

func TestPatternValue_Exact(t *testing.T) {
	t.Parallel()
	data := []byte("test")
	pv := graph.Exact(data)
	if pv.IsWildcard() {
		t.Error("expected IsWildcard to be false")
	}
	if !pv.IsExact() {
		t.Error("expected IsExact to be true")
	}
	if pv.IsBinding() {
		t.Error("expected IsBinding to be false")
	}
	if !bytes.Equal(pv.Data(), data) {
		t.Errorf("expected Data() = %q, got %q", data, pv.Data())
	}
	if !bytes.Equal(pv.ToInterface().([]byte), data) {
		t.Error("expected ToInterface to return data")
	}
}

func TestPatternValue_ExactString(t *testing.T) {
	t.Parallel()
	pv := graph.ExactString("hello")
	if !bytes.Equal(pv.Data(), []byte("hello")) {
		t.Errorf("expected Data() = %q, got %q", "hello", pv.Data())
	}
}

func TestPatternValue_Binding(t *testing.T) {
	t.Parallel()
	pv := graph.Binding("x")
	if pv.IsWildcard() {
		t.Error("expected IsWildcard to be false")
	}
	if pv.IsExact() {
		t.Error("expected IsExact to be false")
	}
	if !pv.IsBinding() {
		t.Error("expected IsBinding to be true")
	}
	if pv.VariableName() != "x" {
		t.Errorf("expected graph.VariableName = %q, got %q", "x", pv.VariableName())
	}
	v, ok := pv.ToInterface().(*graph.Variable)
	if !ok || v.Name != "x" {
		t.Error("expected ToInterface to return *graph.Variable with name 'x'")
	}
}

func TestTypedPattern_ToPattern(t *testing.T) {
	t.Parallel()
	tp := &graph.TypedPattern{
		Subject:   graph.ExactString("alice"),
		Predicate: graph.ExactString("knows"),
		Object:    graph.Binding("friend"),
		Limit:     10,
		Offset:    5,
		Reverse:   true,
	}

	p := tp.ToPattern()

	if !bytes.Equal(p.Subject.([]byte), []byte("alice")) {
		t.Error("expected Subject to be 'alice'")
	}
	if !bytes.Equal(p.Predicate.([]byte), []byte("knows")) {
		t.Error("expected Predicate to be 'knows'")
	}
	v, ok := p.Object.(*graph.Variable)
	if !ok || v.Name != "friend" {
		t.Error("expected Object to be graph.Variable 'friend'")
	}
	if p.Limit != 10 {
		t.Errorf("expected Limit = 10, got %d", p.Limit)
	}
	if p.Offset != 5 {
		t.Errorf("expected Offset = 5, got %d", p.Offset)
	}
	if !p.Reverse {
		t.Error("expected Reverse to be true")
	}
}

func TestTypedPattern_WithWildcard(t *testing.T) {
	t.Parallel()
	tp := graph.NewTypedPattern(
		graph.ExactString("alice"),
		graph.Wildcard(),
		graph.Binding("obj"),
	)

	p := tp.ToPattern()

	if !bytes.Equal(p.Subject.([]byte), []byte("alice")) {
		t.Error("expected Subject to be 'alice'")
	}
	if p.Predicate != nil {
		t.Error("expected Predicate to be nil (wildcard)")
	}
	if v, ok := p.Object.(*graph.Variable); !ok || v.Name != "obj" {
		t.Error("expected Object to be graph.Variable 'obj'")
	}
}

func TestVarAlias(t *testing.T) {
	t.Parallel()
	// Test that graph.Var is an alias for graph.Variable
	var v graph.Var = graph.Var{Name: "test"}
	if v.Name != "test" {
		t.Errorf("expected Name = 'test', got %q", v.Name)
	}

	// Test that *graph.Var works with IsVariable
	pv := &v
	if !graph.IsVariable(pv) {
		t.Error("expected IsVariable to return true for *graph.Var")
	}
}
