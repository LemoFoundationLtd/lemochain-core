// Copyright 2014 The lemochain-core Authors
// This file is part of the lemochain-core library.
//
// The lemochain-core library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lemochain-core library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lemochain-core library. If not, see <http://www.gnu.org/licenses/>.

package store

//
// var OpenFileLimit = 64
//
// type LDBDatabase struct {
// 	beansdb *BeansDB
// }
//
// func (db *LDBDatabase) NewBatch() Batch {
// 	return &LmDBBatch{
// 		db:    db,
// 		items: make([]*BatchItem, 0),
// 		size:  0,
// 	}
//
// }
//
// func (db *LDBDatabase) Put(flg uint32, key, value []byte) error {
// 	return db.beansdb.Put(flg, key, value)
// }
//
// func (db *LDBDatabase) Get(flg uint32, key []byte) ([]byte, error) {
// 	val, err := db.beansdb.Get(flg, key)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if val == nil {
// 		return nil, ErrNotExist
// 	} else {
// 		return val, nil
// 	}
// }
//
// func (db *LDBDatabase) Has(flg uint32, key []byte) (bool, error) {
// 	val, err := db.beansdb.Get(flg, key)
// 	if err != nil {
// 		return false, err
// 	}
//
// 	if val == nil {
// 		return false, nil
// 	} else {
// 		return true, nil
// 	}
// }
//
// func (db *LDBDatabase) Delete(flg uint32, key []byte) error {
// 	panic("implement me")
// }
//
// // NewLDBDatabase returns a LevelDB wrapped object.
// func NewLDBDatabase(beansdb *BeansDB) *LDBDatabase {
// 	return &LDBDatabase{beansdb: beansdb}
// }
//
// func (db *LDBDatabase) Commit(batch Batch) error {
// 	return db.beansdb.Commit(batch)
// }
//
// func (db *LDBDatabase) Close() {
// 	// nil
// }
