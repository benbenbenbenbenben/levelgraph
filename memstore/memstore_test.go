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

package memstore

import (
	"bytes"
	"sync"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
	if m.data == nil {
		t.Error("data map not initialized")
	}
	if m.closed {
		t.Error("new store should not be closed")
	}
}

func TestPutGet(t *testing.T) {
	m := New()
	defer m.Close()

	key := []byte("testkey")
	value := []byte("testvalue")

	// Put a value
	if err := m.Put(key, value, nil); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get the value back
	got, err := m.Get(key, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Errorf("Get returned %q, want %q", got, value)
	}

	// Verify it's a copy (mutation doesn't affect stored value)
	got[0] = 'X'
	got2, _ := m.Get(key, nil)
	if got2[0] == 'X' {
		t.Error("Get should return a copy, not the original")
	}
}

func TestGetNotFound(t *testing.T) {
	m := New()
	defer m.Close()

	_, err := m.Get([]byte("nonexistent"), nil)
	if err != leveldb.ErrNotFound {
		t.Errorf("Get nonexistent key returned %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	m := New()
	defer m.Close()

	key := []byte("deletekey")
	value := []byte("deletevalue")

	m.Put(key, value, nil)

	// Delete the key
	if err := m.Delete(key, nil); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err := m.Get(key, nil)
	if err != leveldb.ErrNotFound {
		t.Errorf("Get after delete returned %v, want ErrNotFound", err)
	}

	// Delete nonexistent key should not error
	if err := m.Delete([]byte("nonexistent"), nil); err != nil {
		t.Errorf("Delete nonexistent key returned %v, want nil", err)
	}
}

func TestBatchWrite(t *testing.T) {
	m := New()
	defer m.Close()

	batch := new(leveldb.Batch)
	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))
	batch.Delete([]byte("key2"))

	if err := m.Write(batch, nil); err != nil {
		t.Fatalf("Write batch failed: %v", err)
	}

	// Check key1 exists
	v1, err := m.Get([]byte("key1"), nil)
	if err != nil {
		t.Errorf("Get key1 failed: %v", err)
	}
	if !bytes.Equal(v1, []byte("value1")) {
		t.Errorf("key1 = %q, want %q", v1, "value1")
	}

	// Check key2 was deleted
	_, err = m.Get([]byte("key2"), nil)
	if err != leveldb.ErrNotFound {
		t.Errorf("key2 should be deleted, got err: %v", err)
	}

	// Check key3 exists
	v3, err := m.Get([]byte("key3"), nil)
	if err != nil {
		t.Errorf("Get key3 failed: %v", err)
	}
	if !bytes.Equal(v3, []byte("value3")) {
		t.Errorf("key3 = %q, want %q", v3, "value3")
	}
}

func TestClose(t *testing.T) {
	m := New()

	m.Put([]byte("key"), []byte("value"), nil)

	if err := m.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations on closed store should fail
	_, err := m.Get([]byte("key"), nil)
	if err != leveldb.ErrClosed {
		t.Errorf("Get on closed store returned %v, want ErrClosed", err)
	}

	err = m.Put([]byte("key"), []byte("value"), nil)
	if err != leveldb.ErrClosed {
		t.Errorf("Put on closed store returned %v, want ErrClosed", err)
	}

	err = m.Delete([]byte("key"), nil)
	if err != leveldb.ErrClosed {
		t.Errorf("Delete on closed store returned %v, want ErrClosed", err)
	}

	batch := new(leveldb.Batch)
	batch.Put([]byte("key"), []byte("value"))
	err = m.Write(batch, nil)
	if err != leveldb.ErrClosed {
		t.Errorf("Write on closed store returned %v, want ErrClosed", err)
	}
}

func TestIterator(t *testing.T) {
	m := New()
	defer m.Close()

	// Insert some data
	m.Put([]byte("a"), []byte("1"), nil)
	m.Put([]byte("b"), []byte("2"), nil)
	m.Put([]byte("c"), []byte("3"), nil)
	m.Put([]byte("d"), []byte("4"), nil)

	t.Run("iterate all", func(t *testing.T) {
		it := m.NewIterator(nil, nil)
		defer it.Release()

		var keys []string
		for it.First(); it.Valid(); it.Next() {
			keys = append(keys, string(it.Key()))
		}
		if len(keys) != 4 {
			t.Errorf("got %d keys, want 4", len(keys))
		}
		// Keys should be sorted
		expected := []string{"a", "b", "c", "d"}
		for i, k := range keys {
			if k != expected[i] {
				t.Errorf("key[%d] = %q, want %q", i, k, expected[i])
			}
		}
	})

	t.Run("iterate range", func(t *testing.T) {
		it := m.NewIterator(&util.Range{Start: []byte("b"), Limit: []byte("d")}, nil)
		defer it.Release()

		var keys []string
		for it.First(); it.Valid(); it.Next() {
			keys = append(keys, string(it.Key()))
		}
		expected := []string{"b", "c"}
		if len(keys) != len(expected) {
			t.Errorf("got %d keys, want %d", len(keys), len(expected))
		}
		for i, k := range keys {
			if k != expected[i] {
				t.Errorf("key[%d] = %q, want %q", i, k, expected[i])
			}
		}
	})

	t.Run("First and Last", func(t *testing.T) {
		it := m.NewIterator(nil, nil)
		defer it.Release()

		if !it.First() {
			t.Error("First() returned false")
		}
		if string(it.Key()) != "a" {
			t.Errorf("First key = %q, want %q", it.Key(), "a")
		}

		if !it.Last() {
			t.Error("Last() returned false")
		}
		if string(it.Key()) != "d" {
			t.Errorf("Last key = %q, want %q", it.Key(), "d")
		}
	})

	t.Run("Seek", func(t *testing.T) {
		it := m.NewIterator(nil, nil)
		defer it.Release()

		if !it.Seek([]byte("b")) {
			t.Error("Seek(b) returned false")
		}
		if string(it.Key()) != "b" {
			t.Errorf("Seek(b) key = %q, want %q", it.Key(), "b")
		}

		// Seek to nonexistent key should find next
		if !it.Seek([]byte("bc")) {
			t.Error("Seek(bc) returned false")
		}
		if string(it.Key()) != "c" {
			t.Errorf("Seek(bc) key = %q, want %q", it.Key(), "c")
		}

		// Seek past end
		if it.Seek([]byte("z")) {
			t.Error("Seek(z) should return false")
		}
	})

	t.Run("Prev", func(t *testing.T) {
		it := m.NewIterator(nil, nil)
		defer it.Release()

		it.Last()
		if !it.Prev() {
			t.Error("Prev from last should return true")
		}
		if string(it.Key()) != "c" {
			t.Errorf("Prev from last key = %q, want %q", it.Key(), "c")
		}
	})

	t.Run("empty iterator", func(t *testing.T) {
		empty := New()
		defer empty.Close()

		it := empty.NewIterator(nil, nil)
		defer it.Release()

		if it.First() {
			t.Error("First on empty iterator should return false")
		}
		if it.Last() {
			t.Error("Last on empty iterator should return false")
		}
		if it.Valid() {
			t.Error("empty iterator should not be valid")
		}
	})
}

func TestIteratorOnClosedStore(t *testing.T) {
	m := New()
	m.Put([]byte("key"), []byte("value"), nil)
	m.Close()

	it := m.NewIterator(nil, nil)
	if it.First() {
		t.Error("First on closed store iterator should return false")
	}
	if it.Error() != leveldb.ErrClosed {
		t.Errorf("Error() = %v, want ErrClosed", it.Error())
	}
}

func TestIteratorKeyValueOnInvalid(t *testing.T) {
	m := New()
	defer m.Close()

	it := m.NewIterator(nil, nil)
	defer it.Release()

	// Before First() is called
	if it.Key() != nil {
		t.Error("Key() on invalid iterator should return nil")
	}
	if it.Value() != nil {
		t.Error("Value() on invalid iterator should return nil")
	}
}

func TestIteratorRelease(t *testing.T) {
	m := New()
	defer m.Close()

	m.Put([]byte("key"), []byte("value"), nil)

	it := m.NewIterator(nil, nil)
	it.First()
	if !it.Valid() {
		t.Error("iterator should be valid before release")
	}

	it.Release()
	if it.Valid() {
		t.Error("iterator should not be valid after release")
	}

	// SetReleaser is a no-op but shouldn't panic
	it.SetReleaser(nil)
}

func TestConcurrentAccess(t *testing.T) {
	m := New()
	defer m.Close()

	var wg sync.WaitGroup
	n := 100

	// Concurrent writes
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := []byte{byte(i)}
			value := []byte{byte(i * 2)}
			m.Put(key, value, nil)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := []byte{byte(i)}
			m.Get(key, nil)
		}(i)
	}
	wg.Wait()

	// Concurrent mixed operations
	for i := 0; i < n; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			m.Put([]byte{byte(i)}, []byte{byte(i)}, nil)
		}(i)
		go func(i int) {
			defer wg.Done()
			m.Get([]byte{byte(i)}, nil)
		}(i)
		go func(i int) {
			defer wg.Done()
			it := m.NewIterator(nil, nil)
			it.First()
			it.Release()
		}(i)
	}
	wg.Wait()
}

func TestPutMakesCopy(t *testing.T) {
	m := New()
	defer m.Close()

	key := []byte("key")
	value := []byte("original")

	m.Put(key, value, nil)

	// Mutate the original
	value[0] = 'X'

	got, _ := m.Get(key, nil)
	if got[0] == 'X' {
		t.Error("Put should store a copy, not the original")
	}
}
