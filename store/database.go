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

var OpenFileLimit = 64

type LDBDatabase struct {
	beansdb *BeansDB
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(beansdb *BeansDB) *LDBDatabase {
	return &LDBDatabase{beansdb: beansdb}
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	return db.beansdb.Put(CACHE_FLG_TRIE, key, key, value)
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	val, err := db.beansdb.Get(key)
	if err != nil {
		return false, err
	}

	if val == nil {
		return false, nil
	} else {
		return true, nil
	}
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	val, err := db.beansdb.Get(key)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, ErrNotExist
	} else {
		return val, nil
	}
}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
	return nil
}

func (db *LDBDatabase) NewBatch(route []byte) Batch {
	batch := &LmDBBatch{
		db:    db,
		items: make([]*BatchItem, 0),
		size:  0,
	}

	batch.route = make([]byte, len(route))
	copy(batch.route, route)
	return batch
}

func (db *LDBDatabase) Commit(batch Batch) error {
	return db.beansdb.Commit(batch)
}

func (db *LDBDatabase) Close() {
	// nil
}
