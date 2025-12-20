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

// Package luxical provides an adapter for the Luxical text embedding model
// to work with LevelGraph's vector search capabilities.
//
// This package wraps the github.com/benbenbenbenbenben/luxical-one-go embedder
// to implement the levelgraph.Embedder interface, enabling semantic search
// on graph data.
//
// # Basic Usage
//
//	import (
//	    "github.com/benbenbenbenbenben/levelgraph"
//	    "github.com/benbenbenbenbenben/levelgraph/vector"
//	    "github.com/benbenbenbenbenben/levelgraph/vector/luxical"
//	)
//
//	// Load the embedder from model files
//	embedder, err := luxical.NewEmbedder("./models/luxical")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer embedder.Close()
//
//	// Create a vector index with the correct dimensions
//	index := vector.NewHNSWIndex(embedder.Dimensions())
//
//	// Open the database with vector support
//	db, err := levelgraph.Open("/path/to/db",
//	    levelgraph.WithVectors(index),
//	    levelgraph.WithAutoEmbed(embedder, levelgraph.AutoEmbedObjects),
//	)
//
// # Int8 Quantization
//
// For reduced memory usage (~50% smaller), use the int8 quantized model:
//
//	embedder, err := luxical.NewEmbedderInt8("./models/luxical-int8")
package luxical

import (
	"fmt"

	lux "github.com/benbenbenbenbenben/luxical-one-go"
)

// Embedder wraps the Luxical text embedding model to implement
// the levelgraph.Embedder interface.
type Embedder struct {
	inner *lux.Embedder
}

// NewEmbedder creates a new Luxical embedder from a directory containing
// the exported model files.
//
// Required files in the directory:
//   - model_config.json
//   - vocab.json
//   - ngram_hash_map.bin
//   - idf_values_f16.bin
//   - sparse_projection_f16.bin
//   - dense_layer_1_f16.bin, dense_layer_2_f16.bin, dense_layer_3_f16.bin
func NewEmbedder(modelDir string) (*Embedder, error) {
	inner, err := lux.LoadFromDirectory(modelDir)
	if err != nil {
		return nil, fmt.Errorf("luxical: failed to load model: %w", err)
	}
	return &Embedder{inner: inner}, nil
}

// NewEmbedderInt8 creates a new Luxical embedder using int8 quantized weights.
// This reduces memory usage by ~50% compared to float16, with minimal accuracy loss.
//
// Required files in the directory:
//   - model_config.json
//   - vocab.json
//   - ngram_hash_map.bin
//   - idf_values_i8.bin
//   - sparse_projection_i8.bin
//   - dense_layer_1_i8.bin, dense_layer_2_i8.bin, dense_layer_3_i8.bin
func NewEmbedderInt8(modelDir string) (*Embedder, error) {
	inner, err := lux.LoadFromDirectoryInt8(modelDir)
	if err != nil {
		return nil, fmt.Errorf("luxical: failed to load int8 model: %w", err)
	}
	return &Embedder{inner: inner}, nil
}

// Embed converts a single text string to a vector embedding.
// The resulting vector has 192 dimensions for the default Luxical model.
func (e *Embedder) Embed(text string) ([]float32, error) {
	vec, err := e.inner.Embed(text)
	if err != nil {
		return nil, fmt.Errorf("luxical: embed failed: %w", err)
	}
	return vec, nil
}

// EmbedBatch converts multiple texts to vector embeddings.
// This processes texts sequentially; the Luxical model does not
// benefit from batching but the interface is provided for convenience.
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	vecs, err := e.inner.EmbedBatch(texts)
	if err != nil {
		return nil, fmt.Errorf("luxical: embed batch failed: %w", err)
	}
	return vecs, nil
}

// Dimensions returns the dimensionality of the embeddings.
// For the default Luxical model, this is 192.
func (e *Embedder) Dimensions() int {
	return e.inner.EmbeddingDim()
}

// Close releases resources associated with the embedder.
func (e *Embedder) Close() error {
	return e.inner.Close()
}

// CosineSimilarity computes the cosine similarity between two embeddings.
// This is a convenience wrapper around the Luxical package's implementation.
// Returns a value between -1 and 1, where 1 means identical direction.
func CosineSimilarity(a, b []float32) float32 {
	return lux.CosineSimilarity(a, b)
}
