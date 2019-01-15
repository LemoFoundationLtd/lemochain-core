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

// Code using batches should try to add this much data to the batch.
// The value was determined empirically.
const IdealBatchSize = 100 * 1024

type Commit interface {
	Commit(batch Batch) error
}

type NewBatch interface {
	NewBatch(route []byte) Batch
}

// Batch is a write-only database that commits changes to its host database
// when Write is called. Batch cannot be used concurrently.
type Batch interface {
	Put(flg uint, key, value []byte) error
	Commit() error
	Route() []byte
	Items() []*BatchItem
	ValueSize() int
	Reset()
}

// Database wraps all database operations. All methods are safe for concurrent use.
type Database interface {
	Put(key, value []byte) error
	NewBatch
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	Close()
}

type BatchItem struct {
	Flg uint
	Key []byte
	Val []byte
}

type LmDBBatch struct {
	db    Commit
	route []byte
	items []*BatchItem
	size  int
}

func (batch *LmDBBatch) Put(flg uint, key, value []byte) error {
	item := &BatchItem{
		Flg: flg,
		Key: key,
		Val: value,
	}
	batch.items = append(batch.items, item)
	batch.size = batch.size + len(value)
	return nil
}

func (batch *LmDBBatch) Commit() error {
	return batch.db.Commit(batch)
}

func (batch *LmDBBatch) Route() []byte {
	return batch.route
}

func (batch *LmDBBatch) Items() []*BatchItem {
	return batch.items
}

func (batch *LmDBBatch) ValueSize() int {
	return batch.size
}

func (batch *LmDBBatch) Reset() {
	batch.size = 0
	batch.items = make([]*BatchItem, 0)
}
