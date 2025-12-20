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
	"context"
	"errors"

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
	"github.com/benbenbenbenbenben/levelgraph/pkg/index"
)

var (
	// facetPrefix is the prefix for component facets
	facetPrefix = []byte("facet::")

	// tripleFacetPrefix is the prefix for triple-level facets
	tripleFacetPrefix = []byte("triple_facet::")

	// ErrFacetsDisabled is returned when facets operations are called but facets are not enabled.
	ErrFacetsDisabled = errors.New("levelgraph: facets are not enabled")
)

// FacetType represents the type of component a facet is attached to.
type FacetType string

const (
	// FacetSubject is a facet on a subject value
	FacetSubject FacetType = "subject"
	// FacetPredicate is a facet on a predicate value
	FacetPredicate FacetType = "predicate"
	// FacetObject is a facet on an object value
	FacetObject FacetType = "object"
)

// genFacetKey generates a key for a component facet.
// Format: facet::<type>::<value>::<key>
func genFacetKey(facetType FacetType, value []byte, key []byte) []byte {
	var buf bytes.Buffer
	buf.Write(facetPrefix)
	buf.WriteString(string(facetType))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(value))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(key))
	return buf.Bytes()
}

// genFacetPrefix generates a prefix for iterating facets on a component.
// Format: facet::<type>::<value>::
func genFacetPrefix(facetType FacetType, value []byte) []byte {
	var buf bytes.Buffer
	buf.Write(facetPrefix)
	buf.WriteString(string(facetType))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(value))
	buf.Write(index.KeySeparator)
	return buf.Bytes()
}

// genTripleFacetKey generates a key for a triple-level facet.
// Format: triple_facet::<spo_key>::<facet_key>
func genTripleFacetKey(triple *graph.Triple, key []byte) []byte {
	var buf bytes.Buffer
	buf.Write(tripleFacetPrefix)
	buf.Write(index.Escape(triple.Subject))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(triple.Predicate))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(triple.Object))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(key))
	return buf.Bytes()
}

// genTripleFacetPrefix generates a prefix for iterating facets on a triple.
func genTripleFacetPrefix(triple *graph.Triple) []byte {
	var buf bytes.Buffer
	buf.Write(tripleFacetPrefix)
	buf.Write(index.Escape(triple.Subject))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(triple.Predicate))
	buf.Write(index.KeySeparator)
	buf.Write(index.Escape(triple.Object))
	buf.Write(index.KeySeparator)
	return buf.Bytes()
}

// SetFacet sets a facet on a component (subject, predicate, or object value).
// The facet is a key-value pair attached to the component.
func (db *DB) SetFacet(ctx context.Context, facetType FacetType, value []byte, key []byte, facetValue []byte) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return ErrClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return ErrFacetsDisabled
	}

	dbKey := genFacetKey(facetType, value, key)
	return db.store.Put(dbKey, facetValue, nil)
}

// GetFacet retrieves a facet from a component.
func (db *DB) GetFacet(ctx context.Context, facetType FacetType, value []byte, key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return nil, ErrFacetsDisabled
	}

	dbKey := genFacetKey(facetType, value, key)
	result, err := db.store.Get(dbKey, nil)
	if err == ErrNotFound {
		return nil, nil
	}
	return result, err
}

// GetFacets retrieves all facets from a component.
// Returns a map of facet keys to values.
func (db *DB) GetFacets(ctx context.Context, facetType FacetType, value []byte) (map[string][]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return nil, ErrFacetsDisabled
	}

	prefix := genFacetPrefix(facetType, value)
	upperBound := append(prefix, 0xFF)

	iter := db.store.NewIterator(&Range{Start: prefix, Limit: upperBound}, nil)
	defer iter.Release()

	result := make(map[string][]byte)
	prefixLen := len(prefix)

	for iter.Next() {
		// Extract the facet key from the database key
		fullKey := iter.Key()
		if len(fullKey) > prefixLen {
			facetKey := index.Unescape(fullKey[prefixLen:])
			facetValue := make([]byte, len(iter.Value()))
			copy(facetValue, iter.Value())
			result[string(facetKey)] = facetValue
		}
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return result, nil
}

// DelFacet deletes a facet from a component.
func (db *DB) DelFacet(ctx context.Context, facetType FacetType, value []byte, key []byte) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return ErrClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return ErrFacetsDisabled
	}

	dbKey := genFacetKey(facetType, value, key)
	return db.store.Delete(dbKey, nil)
}

// SetTripleFacet sets a facet on an entire triple relationship.
func (db *DB) SetTripleFacet(ctx context.Context, triple *graph.Triple, key []byte, value []byte) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return ErrClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return ErrFacetsDisabled
	}

	dbKey := genTripleFacetKey(triple, key)
	return db.store.Put(dbKey, value, nil)
}

// GetTripleFacet retrieves a facet from a triple.
func (db *DB) GetTripleFacet(ctx context.Context, triple *graph.Triple, key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return nil, ErrFacetsDisabled
	}

	dbKey := genTripleFacetKey(triple, key)
	result, err := db.store.Get(dbKey, nil)
	if err == ErrNotFound {
		return nil, nil
	}
	return result, err
}

// GetTripleFacets retrieves all facets from a triple.
func (db *DB) GetTripleFacets(ctx context.Context, triple *graph.Triple) (map[string][]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return nil, ErrFacetsDisabled
	}

	prefix := genTripleFacetPrefix(triple)
	upperBound := append(prefix, 0xFF)

	iter := db.store.NewIterator(&Range{Start: prefix, Limit: upperBound}, nil)
	defer iter.Release()

	result := make(map[string][]byte)
	prefixLen := len(prefix)

	for iter.Next() {
		fullKey := iter.Key()
		if len(fullKey) > prefixLen {
			facetKey := index.Unescape(fullKey[prefixLen:])
			facetValue := make([]byte, len(iter.Value()))
			copy(facetValue, iter.Value())
			result[string(facetKey)] = facetValue
		}
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return result, nil
}

// DelTripleFacet deletes a facet from a triple.
func (db *DB) DelTripleFacet(ctx context.Context, triple *graph.Triple, key []byte) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return ErrClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return ErrFacetsDisabled
	}

	dbKey := genTripleFacetKey(triple, key)
	return db.store.Delete(dbKey, nil)
}

// DelAllTripleFacets deletes all facets from a triple.
func (db *DB) DelAllTripleFacets(ctx context.Context, triple *graph.Triple) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return ErrClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return ErrFacetsDisabled
	}

	prefix := genTripleFacetPrefix(triple)
	upperBound := append(prefix, 0xFF)

	iter := db.store.NewIterator(&Range{Start: prefix, Limit: upperBound}, nil)
	defer iter.Release()

	batch := NewBatch()
	for iter.Next() {
		keyCopy := make([]byte, len(iter.Key()))
		copy(keyCopy, iter.Key())
		batch.Delete(keyCopy)
	}

	if err := iter.Error(); err != nil {
		return err
	}

	return db.store.Write(batch, nil)
}

// FacetIterator iterates over facets on a component or triple.
type FacetIterator struct {
	iter      Iterator
	prefixLen int
}

// GetFacetIterator returns an iterator over facets on a component.
func (db *DB) GetFacetIterator(ctx context.Context, facetType FacetType, value []byte) (*FacetIterator, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return nil, ErrFacetsDisabled
	}

	prefix := genFacetPrefix(facetType, value)
	upperBound := append(prefix, 0xFF)

	iter := db.store.NewIterator(&Range{Start: prefix, Limit: upperBound}, nil)
	return &FacetIterator{
		iter:      iter,
		prefixLen: len(prefix),
	}, nil
}

// GetTripleFacetIterator returns an iterator over facets on a triple.
func (db *DB) GetTripleFacetIterator(ctx context.Context, triple *graph.Triple) (*FacetIterator, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !db.options.FacetsEnabled {
		return nil, ErrFacetsDisabled
	}

	prefix := genTripleFacetPrefix(triple)
	upperBound := append(prefix, 0xFF)

	iter := db.store.NewIterator(&Range{Start: prefix, Limit: upperBound}, nil)
	return &FacetIterator{
		iter:      iter,
		prefixLen: len(prefix),
	}, nil
}

// Next advances the iterator.
func (fi *FacetIterator) Next() bool {
	return fi.iter.Next()
}

// Key returns the current facet key.
func (fi *FacetIterator) Key() []byte {
	fullKey := fi.iter.Key()
	if len(fullKey) > fi.prefixLen {
		return index.Unescape(fullKey[fi.prefixLen:])
	}
	return nil
}

// Value returns the current facet value.
func (fi *FacetIterator) Value() []byte {
	return fi.iter.Value()
}

// Close releases the iterator.
func (fi *FacetIterator) Close() {
	fi.iter.Release()
}

// Error returns any error from the iterator.
func (fi *FacetIterator) Error() error {
	return fi.iter.Error()
}
