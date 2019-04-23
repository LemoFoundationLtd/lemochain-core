package store

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"path/filepath"
)

type WriteExtend interface {
	After(flg uint32, key []byte, val []byte) error
}

type SyncFileDB struct {
	Home string

	Height   uint
	BitCasks []*BitCask

	LevelDB *leveldb.LevelDBDatabase
	Extend  WriteExtend

	DoneChan  chan *Inject
	ErrChan   chan *Inject
	WriteChan chan *Inject
	Quit      chan struct{}
}

func (db *SyncFileDB) path(index int) string {
	dataPathModule := filepath.Join(db.Home, "/%02d/%02d/")
	return fmt.Sprintf(dataPathModule, index>>4, index&0xf)
}

func NewSyncFileDB(home string, levelDB *leveldb.LevelDBDatabase, doneChan chan *Inject, errChan chan *Inject, quit chan struct{}, extend WriteExtend) *SyncFileDB {
	return &SyncFileDB{
		Home:    home,
		Height:  2,
		LevelDB: levelDB,
		Extend:  extend,

		DoneChan:  doneChan,
		ErrChan:   errChan,
		WriteChan: make(chan *Inject, 1024*256),
		Quit:      quit,
	}
}

func (db *SyncFileDB) newBitCask(index int) *BitCask {
	path := db.path(index)
	bitCask, err := NewBitCask(path, index, db.LevelDB)
	if err != nil {
		panic("create bit cask err: " + err.Error())
	} else {
		return bitCask
	}
}

func (db *SyncFileDB) Open() {
	count := 1 << (uint(db.Height) * 4)
	db.BitCasks = make([]*BitCask, count)

	for index := 0; index < count; index++ {
		db.BitCasks[index] = db.newBitCask(index)
	}

	go db.start(db.DoneChan, db.ErrChan)
}

func (db *SyncFileDB) start(Done chan *Inject, Err chan *Inject) {
	for {
		select {
		case <-db.Quit:
			return
		case writeOp := <-db.WriteChan:
			err := db.put(writeOp.Flg, writeOp.Key, writeOp.Val)
			if err != nil {
				log.Errorf("bitcask put data err: %s, flg: %d, key: %s", err.Error(), writeOp.Flg, common.ToHex(writeOp.Key))
				Err <- writeOp
			} else {
				db.afterWriteExtend(writeOp)
				Done <- writeOp
			}
		}
	}
}

func (db *SyncFileDB) Get(flag uint32, key []byte) ([]byte, error) {
	bitcask := db.route(key)
	if bitcask == nil {
		panic(fmt.Sprintf("queue get k/v. bitcask is nil.key: %s", common.ToHex(key)))
	} else {
		return bitcask.Get(flag, key)
	}
}

func (db *SyncFileDB) Put(flag uint32, key []byte, val []byte) {
	op := &Inject{
		Flg: flag,
		Key: key,
		Val: val,
	}

	select {
	case <-db.Quit:
		return
	case db.WriteChan <- op:
		return
	default:
		log.Errorf("channel queue is busy!!")
	}
}

func (db *SyncFileDB) put(flag uint32, key []byte, val []byte) error {
	bitcask := db.route(key)
	return bitcask.Put(flag, key, val)
}

func (db *SyncFileDB) afterWriteExtendSuc(op *Inject) {
	// nil
}

func (db *SyncFileDB) afterWriteExtendErr(op *Inject, err error) {
	log.Errorf("write extend data err: " + err.Error())
}

func (db *SyncFileDB) afterWriteExtend(op *Inject) {
	index := 0
	for ; index < 1024; index++ {
		err := db.Extend.After(op.Flg, op.Key, op.Val)
		if err != nil {
			db.afterWriteExtendErr(op, err)
			continue
		} else {
			db.afterWriteExtendSuc(op)
			break
		}
	}

	if index == 1024 {
		panic("write extend data err !!!")
	}
}

func (db *SyncFileDB) route(key []byte) *BitCask {
	num := Byte2Uint32(key)
	index := num >> ((8 - db.Height) * 4)
	return db.BitCasks[index]
}
