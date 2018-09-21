// Copyright 2014 The lemochain-go Authors
// This file is part of the lemochain-go library.
//
// The lemochain-go library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lemochain-go library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lemochain-go library. If not, see <http://www.gnu.org/licenses/>.

package store

import (
	"sync"
)

var OpenFileLimit = 64

type LDBDatabase struct {
	db *LmDataBase

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
}

func (db *LDBDatabase) Close() {
	// Stop the metrics collection to avoid internal database races
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			//db.log.Error("Metrics collection failed", "err", err)
		}
	}
	err := db.db.Close()
	if err == nil {
		//db.log.Info("TrieDatabase closed")
	} else {
		//db.log.Error("Failed to close database", "err", err)
	}
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(database *LmDataBase, cache int, handles int) *LDBDatabase {
	//logger := log.New("database", file)

	// Ensure we have some minimal caching and file guarantees
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	//logger.Info("Allocated cache and file handles", "cache", cache, "handles", handles)

	// Open the db and recover any potential corruptions
	//db, err := leveldb.OpenFile(file, &opt.Options{
	//	OpenFilesCacheCapacity: handles,
	//	BlockCacheCapacity:     cache / 2 * opt.MiB,
	//	WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
	//	Filter:                 filter.NewBloomFilter(10),
	//})

	//if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
	//	db, err = leveldb.RecoverFile(file, nil)
	//}
	// (Re)check for errors and abort if opening of the db failed
	//if err != nil {
	//	return nil, err
	//}
	return &LDBDatabase{db: database}
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	// Generate the data to write to disk, update the meter and write
	//value = rle.Compress(value)

	return db.db.Put(key, value)
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(key)
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	// Retrieve the key and increment the miss counter if not found
	val, err := db.db.Get(key)
	if err != nil {
		return nil, err
	}
	return val, nil
}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
	// Execute the actual operation
	//return db.db.Delete(key, nil)
	return db.Delete(key)
}

//func (db *LDBDatabase) NewIterator() iterator.Iterator {
//	return db.db.NewIterator(nil, nil)
//}
//
//// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
//func (db *LDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
//	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
//}

func (db *LDBDatabase) NewBatch() Batch {
	return &LDBBatch{
		db: db,
		b:  make([]*BatchItem, 0, 10000),
	}
}

func (db *LDBDatabase) commit(batch *LDBBatch) error {
	items := batch.b
	return db.db.Commit(items)
}

type LDBBatch struct {
	db *LDBDatabase
	b  []*BatchItem
}

func (db *LDBDatabase) LDB() *LmDataBase {
	return db.db
}

func (b *LDBBatch) Put(key, value []byte) error {
	item := &BatchItem{
		Key: key,
		Val: value,
	}
	b.b = append(b.b, item)
	return nil
}

func (b *LDBBatch) Write() error {
	return b.db.commit(b)
}

func (b *LDBBatch) ValueSize() int {
	return len(b.b)
}

func (b *LDBBatch) Reset() {
	b.b = make([]*BatchItem, 0, 10000)
}

//type table struct {
//	db     TrieDatabase
//	prefix string
//}
//
//// NewTable returns a TrieDatabase object that prefixes all keys with a given
//// string.
//func NewTable(db TrieDatabase, prefix string) TrieDatabase {
//	return &table{
//		db:     db,
//		prefix: prefix,
//	}
//}
//
//func (dt *table) Put(key []byte, value []byte) error {
//	return dt.db.Put(append([]byte(dt.prefix), key...), value)
//}
//
//func (dt *table) Has(key []byte) (bool, error) {
//	return dt.db.Has(append([]byte(dt.prefix), key...))
//}
//
//func (dt *table) Get(key []byte) ([]byte, error) {
//	return dt.db.Get(append([]byte(dt.prefix), key...))
//}
//
//func (dt *table) Delete(key []byte) error {
//	return dt.db.Delete(append([]byte(dt.prefix), key...))
//}
//
//func (dt *table) Close() {
//	// Do nothing; don't close the underlying DB.
//}
//
//type tableBatch struct {
//	batch  Batch
//	prefix string
//}
//
//// NewTableBatch returns a Batch object which prefixes all keys with a given string.
//func NewTableBatch(db TrieDatabase, prefix string) Batch {
//	return &tableBatch{db.NewBatch(), prefix}
//}
//
//func (dt *table) NewBatch() Batch {
//	return &tableBatch{dt.db.NewBatch(), dt.prefix}
//}
//
//func (tb *tableBatch) Put(key, value []byte) error {
//	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
//}
//
//func (tb *tableBatch) Write() error {
//	return tb.batch.Write()
//}
//
//func (tb *tableBatch) ValueSize() int {
//	return tb.batch.ValueSize()
//}
//
//func (tb *tableBatch) Reset() {
//	tb.batch.Reset()
//}
