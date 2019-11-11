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

package leveldb

import (
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
	"strings"
	"sync"
	"time"
)

var OpenFileLimit = 64

type LevelDBDatabase struct {
	fn string      // filename for reporting
	db *leveldb.DB // LevelDB instance

	getTimer       gometrics.Timer // 对数据库进行get操作的频率和时间分布情况
	putTimer       gometrics.Timer // 对数据库进行put操作的频率和时间分布情况
	delTimer       gometrics.Timer // 对数据库进行delete操作的频率和时间分布情况
	missMeter      gometrics.Meter // 对数据库进行get操作失败的频率
	readMeter      gometrics.Meter // 对数据库进行get操作之后返回对返回回来的value长度进行标记
	writeMeter     gometrics.Meter // 对数据库进行put操作对放进去的value长度进行标记
	compTimeMeter  gometrics.Meter // Meter for measuring the total time spent in database compaction
	compReadMeter  gometrics.Meter // Meter for measuring the data read during compaction
	compWriteMeter gometrics.Meter // Meter for measuring the data written during compaction

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database

	// log log.Logger // Contextual logger tracking the database path
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLevelDBDatabase(file string, cache int, handles int) *LevelDBDatabase {
	// logger := log.New("database", file)

	// Ensure we have some minimal caching and file guarantees
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	// logger.Info("Allocated cache and file handles", "cache", cache, "handles", handles)

	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})

	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}

	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		panic("new level database err: " + err.Error())
	} else {
		return &LevelDBDatabase{
			fn: file,
			db: db,
			// log: logger,
		}
	}
}

// Path returns the path to the database directory.
func (db *LevelDBDatabase) Path() string {
	return db.fn
}

// Put puts the given key / value to the queue
func (db *LevelDBDatabase) Put(key []byte, value []byte) error {
	// Measure the database put latency, if requested
	if db.putTimer != nil {
		defer db.putTimer.UpdateSince(time.Now())
	}
	// Generate the data to write to disk, update the meter and write
	// value = rle.Compress(value)

	if db.writeMeter != nil {
		db.writeMeter.Mark(int64(len(value)))
	}
	return db.db.Put(key, value, nil)
}

func (db *LevelDBDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (db *LevelDBDatabase) Get(key []byte) ([]byte, error) {
	// Measure the database get latency, if requested
	if db.getTimer != nil {
		defer db.getTimer.UpdateSince(time.Now())
	}
	// Retrieve the key and increment the miss counter if not found
	dat, err := db.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		if db.missMeter != nil {
			db.missMeter.Mark(1)
		}
		return nil, err
	}

	// Otherwise update the actually retrieved amount of data
	if db.readMeter != nil {
		db.readMeter.Mark(int64(len(dat)))
	}

	// return rle.Decompress(dat)
	return dat, nil
}

// Delete deletes the key from the queue and database
func (db *LevelDBDatabase) Delete(key []byte) error {
	// Measure the database delete latency, if requested
	if db.delTimer != nil {
		defer db.delTimer.UpdateSince(time.Now())
	}
	// Execute the actual operation
	return db.db.Delete(key, nil)
}

func (db *LevelDBDatabase) NewIterator() iterator.Iterator {
	return db.db.NewIterator(nil, nil)
}

// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
func (db *LevelDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
}

func (db *LevelDBDatabase) Close() {
	// Stop the metrics collection to avoid internal database races
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			// db.log.Error("Metrics collection failed", "err", err)
		}
	}

	err := db.db.Close()
	if err == nil {
		// db.log.Info("Database closed")
	} else {
		// db.log.Error("Failed to close database", "err", err)
	}
}

func (db *LevelDBDatabase) LDB() *leveldb.DB {
	return db.db
}

// Meter configures the database metrics collectors and
func (db *LevelDBDatabase) Meter() {
	// Short circuit metering if the metrics system is disabled
	if !metrics.Enabled {
		return
	}
	// Initialize all the metrics collector at the requested prefix
	db.getTimer = metrics.NewTimer(metrics.LevelDb_get_timerName)
	db.putTimer = metrics.NewTimer(metrics.LevelDb_put_timerName)
	db.delTimer = metrics.NewTimer(metrics.LevelDb_del_timerName)
	db.missMeter = metrics.NewMeter(metrics.LevelDb_miss_meterName)
	db.readMeter = metrics.NewMeter(metrics.LevelDb_read_meterName)
	db.writeMeter = metrics.NewMeter(metrics.LevelDb_write_meterName)
	db.compTimeMeter = metrics.NewMeter(metrics.LevelDb_compTime_meteName)
	db.compReadMeter = metrics.NewMeter(metrics.LevelDb_compRead_meterName)
	db.compWriteMeter = metrics.NewMeter(metrics.LevelDb_compWrite_meterName)
	// Create a quit channel for the periodic collector and run it
	db.quitLock.Lock()
	db.quitChan = make(chan chan error)
	db.quitLock.Unlock()

	go db.meter(3 * time.Second)
}

// meter periodically retrieves internal leveldb counters and reports them to
// the metrics subsystem.
//
// This is how a stats table look like (currently):
//   Compactions
//    Level |   Tables   |    Size(MB)   |    Time(sec)  |    Read(MB)   |   Write(MB)
//   -------+------------+---------------+---------------+---------------+---------------
//      0   |          0 |       0.00000 |       1.27969 |       0.00000 |      12.31098
//      1   |         85 |     109.27913 |      28.09293 |     213.92493 |     214.26294
//      2   |        523 |    1000.37159 |       7.26059 |      66.86342 |      66.77884
//      3   |        570 |    1113.18458 |       0.00000 |       0.00000 |       0.00000
//
func (db *LevelDBDatabase) meter(refresh time.Duration) {
	// Create the counters to store current and previous values
	counters := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		counters[i] = make([]float64, 3)
	}
	// Iterate ad infinitum and collect the stats
	for i := 1; ; i++ {
		// Retrieve the database stats
		stats, err := db.db.GetProperty("leveldb.stats")
		if err != nil {
			// db.log.Error("Failed to read database stats", "err", err)
			return
		}
		// Find the compaction table, skip the header
		lines := strings.Split(stats, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
			lines = lines[1:]
		}
		if len(lines) <= 3 {
			// db.log.Error("Compaction table not found")
			return
		}
		lines = lines[3:]

		// Iterate over all the table rows, and accumulate the entries
		for j := 0; j < len(counters[i%2]); j++ {
			counters[i%2][j] = 0
		}
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) != 6 {
				break
			}
			for idx, counter := range parts[3:] {
				value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64)
				if err != nil {
					// db.log.Error("Compaction entry parsing failed", "err", err)
					return
				}
				counters[i%2][idx] += value
			}
		}
		// Update all the requested meters
		if db.compTimeMeter != nil {
			db.compTimeMeter.Mark(int64((counters[i%2][0] - counters[(i-1)%2][0]) * 1000 * 1000 * 1000))
		}
		if db.compReadMeter != nil {
			db.compReadMeter.Mark(int64((counters[i%2][1] - counters[(i-1)%2][1]) * 1024 * 1024))
		}
		if db.compWriteMeter != nil {
			db.compWriteMeter.Mark(int64((counters[i%2][2] - counters[(i-1)%2][2]) * 1024 * 1024))
		}
		// Sleep a bit, then repeat the stats collection
		select {
		case errc := <-db.quitChan:
			// Quit requesting, stop hammering the database
			errc <- nil
			return

		case <-time.After(refresh):
			// Timeout, gather a new set of stats
		}
	}
}

// // This is how the iostats look like (currently):
// // Read(MB):3895.04860 Write(MB):3654.64712
// func (db *LevelDBDatabase) meter(refresh time.Duration) {
// 	// Create the counters to store current and previous compaction values
// 	compactions := make([][]float64, 2)
// 	for i := 0; i < 2; i++ {
// 		compactions[i] = make([]float64, 3)
// 	}
// 	// Create storage for iostats.
// 	var iostats [2]float64
// 	// Iterate ad infinitum and collect the stats
// 	for i := 1; ; i++ {
// 		// Retrieve the database stats
// 		stats, err := db.db.GetProperty("leveldb.stats")
// 		if err != nil {
// 			// db.log.Error("Failed to read database stats", "err", err)
// 			return
// 		}
// 		// Find the compaction table, skip the header
// 		lines := strings.Split(stats, "\n")
// 		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
// 			lines = lines[1:]
// 		}
// 		if len(lines) <= 3 {
// 			// db.log.Error("Compaction table not found")
// 			return
// 		}
// 		lines = lines[3:]
//
// 		// Iterate over all the table rows, and accumulate the entries
// 		for j := 0; j < len(compactions[i%2]); j++ {
// 			compactions[i%2][j] = 0
// 		}
// 		for _, line := range lines {
// 			parts := strings.Split(line, "|")
// 			if len(parts) != 6 {
// 				break
// 			}
// 			for idx, counter := range parts[3:] {
// 				value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64)
// 				if err != nil {
// 					// db.log.Error("Compaction entry parsing failed", "err", err)
// 					return
// 				}
// 				compactions[i%2][idx] += value
// 			}
// 		}
// 		// Update all the requested meters
// 		// if db.compTimeMeter != nil {
// 		// 	db.compTimeMeter.Mark(int64((compactions[i%2][0] - compactions[(i-1)%2][0]) * 1000 * 1000 * 1000))
// 		// }
// 		// if db.compReadMeter != nil {
// 		// 	db.compReadMeter.Mark(int64((compactions[i%2][1] - compactions[(i-1)%2][1]) * 1024 * 1024))
// 		// }
// 		// if db.compWriteMeter != nil {
// 		// 	db.compWriteMeter.Mark(int64((compactions[i%2][2] - compactions[(i-1)%2][2]) * 1024 * 1024))
// 		// }
//
// 		// Retrieve the database iostats.
// 		ioStats, err := db.db.GetProperty("leveldb.iostats")
// 		if err != nil {
// 			// db.log.Error("Failed to read database iostats", "err", err)
// 			return
// 		}
// 		parts := strings.Split(ioStats, " ")
// 		if len(parts) < 2 {
// 			// db.log.Error("Bad syntax of ioStats", "ioStats", ioStats)
// 			return
// 		}
// 		r := strings.Split(parts[0], ":")
// 		if len(r) < 2 {
// 			// db.log.Error("Bad syntax of read entry", "entry", parts[0])
// 			return
// 		}
// 		read, err := strconv.ParseFloat(r[1], 64)
// 		if err != nil {
// 			// db.log.Error("Read entry parsing failed", "err", err)
// 			return
// 		}
// 		w := strings.Split(parts[1], ":")
// 		if len(w) < 2 {
// 			// db.log.Error("Bad syntax of write entry", "entry", parts[1])
// 			return
// 		}
// 		write, err := strconv.ParseFloat(w[1], 64)
// 		if err != nil {
// 			// db.log.Error("Write entry parsing failed", "err", err)
// 			return
// 		}
// 		// if db.diskReadMeter != nil {
// 		// 	db.diskReadMeter.Mark(int64((read - iostats[0]) * 1024 * 1024))
// 		// }
// 		// if db.diskWriteMeter != nil {
// 		// 	db.diskWriteMeter.Mark(int64((write - iostats[1]) * 1024 * 1024))
// 		// }
// 		iostats[0] = read
// 		iostats[1] = write
//
// 		// Sleep a bit, then repeat the stats collection
// 		select {
// 		case errc := <-db.quitChan:
// 			// Quit requesting, stop hammering the database
// 			errc <- nil
// 			return
//
// 		case <-time.After(refresh):
// 			// Timeout, gather a new set of stats
// 		}
// 	}
// }

func (db *LevelDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.db, b: new(leveldb.Batch)}
}

type Batch interface {
	DatabasePutter
	ValueSize() int // amount of data in the batch
	Write() error
	// Reset resets the batch for reuse
	Reset()
}

type ldbBatch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

func (b *ldbBatch) Reset() {
	b.b.Reset()
	b.size = 0
}
