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

//go:build !js

package levelgraph

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Iterator is an alias for the leveldb iterator interface.
type Iterator = iterator.Iterator

// Batch is an alias for leveldb.Batch.
type Batch = leveldb.Batch

// Range is an alias for util.Range.
type Range = util.Range

// ReadOptions is an alias for opt.ReadOptions.
type ReadOptions = opt.ReadOptions

// WriteOptions is an alias for opt.WriteOptions.
type WriteOptions = opt.WriteOptions

// NewBatch creates a new batch.
func NewBatch() *Batch {
	return new(leveldb.Batch)
}

// openLevelDB opens a LevelDB database at the given path.
func openLevelDB(path string) (KVStore, error) {
	return leveldb.OpenFile(path, &opt.Options{})
}

// ErrNotFound is returned when a key is not found.
var ErrNotFound = leveldb.ErrNotFound
