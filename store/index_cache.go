package store

import (
	"bytes"
	"encoding/binary"
)

type CacheBucket struct {
	Index   uint32
	MaxCnt  uint32
	UsedCnt uint32
	ItemLen uint8
	UnitCnt uint8
	Buf     []byte
}

func NewCacheBucket(index uint32, maxCnt uint32, itemLen uint8) *CacheBucket {
	return &CacheBucket{
		Index:   index,
		MaxCnt:  maxCnt,
		UsedCnt: 0,
		ItemLen: itemLen,
		UnitCnt: 16,
		Buf:     make([]byte, maxCnt*uint32(itemLen)),
	}
}

func (bucket *CacheBucket) Remind() uint32 {
	return bucket.MaxCnt - bucket.UsedCnt
}

func (bucket *CacheBucket) GetUnit(start uint32) ([]byte, error) {
	if (start < 0) || (start+16 > bucket.MaxCnt) {
		return nil, ErrArgInvalid
	} else {
		startOffset := start * uint32(bucket.ItemLen)
		stopOffset := (start + 16) * uint32(bucket.ItemLen)
		return bucket.Buf[startOffset:stopOffset], nil
	}
}

func (bucket *CacheBucket) SetUnit(buf []byte) (uint32, error) {
	if buf == nil || uint32(len(buf))/uint32(bucket.ItemLen) != 16 {
		return 0, ErrArgInvalid
	}

	if (bucket.Remind() * uint32(bucket.ItemLen)) < uint32(len(buf)) {
		return 0, ErrArgInvalid
	}

	copy(bucket.Buf[bucket.UsedCnt*uint32(bucket.ItemLen):], buf[:16*uint32(bucket.ItemLen)])

	offset := bucket.UsedCnt
	bucket.UsedCnt = bucket.UsedCnt + uint32(bucket.UnitCnt)
	return offset, nil
}

func (bucket *CacheBucket) UpdateUint(start uint32, buf []byte) error {
	if buf == nil || uint32(len(buf))/uint32(bucket.ItemLen) != 16 {
		return ErrArgInvalid
	}

	if start < 0 || start+16 > bucket.MaxCnt {
		return ErrArgInvalid
	}

	copy(bucket.Buf[start*uint32(bucket.ItemLen):], buf[:16*uint32(bucket.ItemLen)])
	return nil
}

type CachePool struct {
	Current uint32
	MaxCnt  uint32
	ItemLen uint8
	Buckets []*CacheBucket
	UintBuf *bytes.Buffer
}

func NewCachePool(bucketMaxCnt uint32, bucketItemLen uint8) *CachePool {
	pool := &CachePool{
		Current: 0,
		MaxCnt:  bucketMaxCnt,
		ItemLen: bucketItemLen,
	}

	buf := make([]byte, 1024*1024*5)
	pool.UintBuf = bytes.NewBuffer(buf)

	pool.Buckets = append(pool.Buckets, NewCacheBucket(pool.Current, pool.MaxCnt, pool.ItemLen))
	return pool
}

func (pool *CachePool) checkPools(size uint32) {
	bucket := pool.Buckets[pool.Current]
	if bucket.Remind() < size/uint32(pool.ItemLen) {
		pool.Current = pool.Current + 1
		pool.Buckets = append(pool.Buckets, NewCacheBucket(pool.Current, pool.MaxCnt, pool.ItemLen))
	}
}

func (pool *CachePool) Get(header uint32, index uint32) ([]byte, error) {
	if index < 0 || index > 16 {
		return nil, ErrArgInvalid
	}

	buf, err := pool.GetUnit(header)
	if err != nil {
		return nil, err
	} else {
		return buf[index*uint32(pool.ItemLen):], nil
	}
}

func (pool *CachePool) GetUnit(header uint32) ([]byte, error) {
	bucketIndex := GetBucketIndex(header)
	bucket := pool.Buckets[bucketIndex]

	itemIndex := GetItemIndex(header)
	return bucket.GetUnit(itemIndex)
}

func (pool *CachePool) SetUnit(buf []byte) (uint32, error) {
	pool.checkPools(uint32(len(buf)))
	bucket := pool.Buckets[pool.Current]
	offset, err := bucket.SetUnit(buf)
	if err != nil {
		return 0, err
	} else {
		return GetPos(pool.Current, offset), nil
	}
}

func (pool *CachePool) UpdateUnit(start uint32, buf []byte) error {
	bucketIndex := GetBucketIndex(start)
	bucket := pool.Buckets[bucketIndex]
	index := GetItemIndex(start)
	return bucket.UpdateUint(index, buf)
}

func (pool *CachePool) Malloc(size uint32) *bytes.Buffer {
	if uint32(pool.UintBuf.Cap()) < size {
		pool.UintBuf = bytes.NewBuffer(make([]byte, size))
	}

	pool.UintBuf.Reset()
	return pool.UintBuf
}

func GetNodes(pool *CachePool, header uint32) ([]*Node, error) {
	buf, err := pool.GetUnit(header)
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, 16)
	for index := 0; index < 16; index++ {
		var node Node
		err = binary.Read(NewLmBuffer(buf[index*int(nodeSize):]), binary.LittleEndian, &node)
		if err != nil {
			return nil, err
		} else {
			nodes[index] = &node
		}
	}

	return nodes, nil
}

func SetNodes(pool *CachePool, nodes []*Node) (uint32, error) {
	if len(nodes) != 16 {
		return 0, ErrArgInvalid
	}

	buf := pool.Malloc(16 * uint32(pool.ItemLen))
	for index := 0; index < 16; index++ {
		node := nodes[index]
		err := binary.Write(buf, binary.LittleEndian, node)
		if err != nil {
			return 0, err
		}
	}

	return pool.SetUnit(buf.Bytes())
}

func UpdateNodes(pool *CachePool, header uint32, nodes []*Node) error {
	if len(nodes) != 16 {
		return ErrArgInvalid
	}

	nodeLen := uint32(binary.Size(Node{}))
	buf := pool.Malloc(16 * nodeLen)
	for index := 0; index < 16; index++ {
		node := nodes[index]
		err := binary.Write(buf, binary.LittleEndian, node)
		if err != nil {
			return err
		}
	}

	return pool.UpdateUnit(header, buf.Bytes())
}

func GetItems(pool *CachePool, header uint32) ([]*TItem, error) {
	buf, err := pool.GetUnit(header)
	if err != nil {
		return nil, err
	}

	items := make([]*TItem, 16)
	for index := 0; index < 16; index++ {
		var item HItem
		start := index * int(pool.ItemLen)
		keyStart := index*int(pool.ItemLen) + int(itemHeaderSize)
		err = binary.Read(NewLmBuffer(buf[start:]), binary.LittleEndian, &item)
		if err != nil {
			return nil, err
		} else {
			key := make([]byte, keySize)
			err = binary.Read(NewLmBuffer(buf[keyStart:]), binary.LittleEndian, key)
			if err != nil {
				return nil, err
			} else {
				items[index] = &TItem{
					Flg: item.Flg,
					Num: item.Num,
					Pos: item.Pos,
					Key: key,
				}
			}
		}
	}

	return items, nil
}

func SetItems(pool *CachePool, items []*TItem) (uint32, error) {
	if len(items) != 16 {
		return 0, ErrArgInvalid
	}

	buf := pool.Malloc(16 * uint32(pool.ItemLen))
	for index := 0; index < 16; index++ {
		titem := items[index]
		key := titem.Key
		if uint32(len(key)) != keySize {
			return 0, ErrArgInvalid
		}

		item := &HItem{
			Flg: titem.Flg,
			Num: titem.Num,
			Pos: titem.Pos,
		}

		err := binary.Write(buf, binary.LittleEndian, item)
		if err != nil {
			return 0, err
		}

		err = binary.Write(buf, binary.LittleEndian, titem.Key)
		if err != nil {
			return 0, err
		}
	}

	return pool.SetUnit(buf.Bytes())
}

func UpdateItems(pool *CachePool, start uint32, items []*TItem) error {
	if len(items) != 16 {
		return ErrArgInvalid
	}

	buf := pool.Malloc(16 * uint32(pool.ItemLen))
	for index := 0; index < 16; index++ {
		titem := items[index]

		key := titem.Key
		if uint32(len(key)) != keySize {
			return ErrArgInvalid
		}

		item := &HItem{
			Flg: titem.Flg,
			Num: titem.Num,
			Pos: titem.Pos,
		}

		err := binary.Write(buf, binary.LittleEndian, item)
		if err != nil {
			return err
		}

		err = binary.Write(buf, binary.LittleEndian, titem.Key)
		if err != nil {
			return err
		}
	}

	return pool.UpdateUnit(start, buf.Bytes())
}
