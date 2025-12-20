// Copyright (c) 2024 LevelGraph Go Contributors
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software
// is furnished to do so, subject to the following conditions:
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

package luxical

import (
	"os"
	"testing"
)

// modelDir is the path to the test model directory.
// Set via environment variable LUXICAL_MODEL_DIR, defaults to luxical-one export.
func getModelDir() string {
	if dir := os.Getenv("LUXICAL_MODEL_DIR"); dir != "" {
		return dir
	}
	// Default to the local development path
	return "/home/ben/luxical-one/export"
}

func TestNewEmbedder(t *testing.T) {
	modelDir := getModelDir()
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		t.Skipf("model directory not found: %s", modelDir)
	}

	embedder, err := NewEmbedder(modelDir)
	if err != nil {
		t.Fatalf("NewEmbedder failed: %v", err)
	}
	defer embedder.Close()

	// Check dimensions
	dims := embedder.Dimensions()
	if dims != 192 {
		t.Errorf("expected 192 dimensions, got %d", dims)
	}
}

func TestNewEmbedderInt8(t *testing.T) {
	modelDir := getModelDir()
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		t.Skipf("model directory not found: %s", modelDir)
	}

	embedder, err := NewEmbedderInt8(modelDir)
	if err != nil {
		t.Fatalf("NewEmbedderInt8 failed: %v", err)
	}
	defer embedder.Close()

	// Check dimensions
	dims := embedder.Dimensions()
	if dims != 192 {
		t.Errorf("expected 192 dimensions, got %d", dims)
	}
}

func TestEmbed(t *testing.T) {
	modelDir := getModelDir()
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		t.Skipf("model directory not found: %s", modelDir)
	}

	embedder, err := NewEmbedder(modelDir)
	if err != nil {
		t.Fatalf("NewEmbedder failed: %v", err)
	}
	defer embedder.Close()

	vec, err := embedder.Embed("hello world")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(vec) != 192 {
		t.Errorf("expected 192-dim vector, got %d", len(vec))
	}

	// Vector should not be all zeros
	allZero := true
	for _, v := range vec {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("embedding is all zeros")
	}
}

func TestEmbedBatch(t *testing.T) {
	modelDir := getModelDir()
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		t.Skipf("model directory not found: %s", modelDir)
	}

	embedder, err := NewEmbedder(modelDir)
	if err != nil {
		t.Fatalf("NewEmbedder failed: %v", err)
	}
	defer embedder.Close()

	texts := []string{"hello", "world", "test"}
	vecs, err := embedder.EmbedBatch(texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(vecs) != 3 {
		t.Errorf("expected 3 embeddings, got %d", len(vecs))
	}

	for i, vec := range vecs {
		if len(vec) != 192 {
			t.Errorf("embedding %d: expected 192-dim, got %d", i, len(vec))
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	modelDir := getModelDir()
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		t.Skipf("model directory not found: %s", modelDir)
	}

	embedder, err := NewEmbedder(modelDir)
	if err != nil {
		t.Fatalf("NewEmbedder failed: %v", err)
	}
	defer embedder.Close()

	// Similar texts should have high similarity
	vec1, _ := embedder.Embed("tennis is a racket sport")
	vec2, _ := embedder.Embed("badminton is a racket sport")
	vec3, _ := embedder.Embed("programming in Go")

	sim12 := CosineSimilarity(vec1, vec2)
	sim13 := CosineSimilarity(vec1, vec3)

	// Tennis and badminton should be more similar than tennis and programming
	if sim12 <= sim13 {
		t.Errorf("expected tennis/badminton sim (%.4f) > tennis/programming sim (%.4f)", sim12, sim13)
	}

	// Similar sports should have high similarity
	if sim12 < 0.5 {
		t.Errorf("expected tennis/badminton similarity > 0.5, got %.4f", sim12)
	}

	t.Logf("Similarities: tennis/badminton=%.4f, tennis/programming=%.4f", sim12, sim13)
}

func TestEmbedderImplementsInterface(t *testing.T) {
	// This test verifies at compile time that Embedder implements the
	// levelgraph.Embedder interface
	var _ interface {
		Embed(text string) ([]float32, error)
		EmbedBatch(texts []string) ([][]float32, error)
		Dimensions() int
	} = (*Embedder)(nil)
}
