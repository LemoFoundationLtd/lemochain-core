package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_newTimeBucket(t *testing.T) {
	timeBuckets := newTimeBucket(0)
	assert.Equal(t, uint32(0), timeBuckets.TimeBase)
	assert.Equal(t, 40, timeBuckets.cap)

	timeBuckets = newTimeBucket(BucketDuration*2 - 1)
	assert.Equal(t, BucketDuration, timeBuckets.TimeBase)

	timeBuckets = newTimeBucket(BucketDuration * 2)
	assert.Equal(t, BucketDuration*2, timeBuckets.TimeBase)

	timeBuckets = newTimeBucket(BucketDuration*2 + 1)
	assert.Equal(t, BucketDuration*2, timeBuckets.TimeBase)
}

func TestTimeBuckets_Add(t *testing.T) {
	timeBuckets := newTimeBucket(BucketDuration * 2)

	// invalid time
	err := timeBuckets.Add(BucketDuration*2-1, common.HexToHash("1"))
	assert.Equal(t, ErrTimeBucketTime, err)

	// success
	err = timeBuckets.Add(BucketDuration*2, common.HexToHash("1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(timeBuckets.buckets[0]))
	assert.Equal(t, common.HexToHash("1"), timeBuckets.buckets[0][0])
	assert.Equal(t, 40, timeBuckets.cap)

	// same bucket, different hash
	err = timeBuckets.Add(BucketDuration*2+1, common.HexToHash("2"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(timeBuckets.buckets[0]))
	assert.Equal(t, common.HexToHash("1"), timeBuckets.buckets[0][0])
	assert.Equal(t, common.HexToHash("2"), timeBuckets.buckets[0][1])
	assert.Equal(t, 40, timeBuckets.cap)

	// different bucket
	err = timeBuckets.Add(BucketDuration*3, common.HexToHash("3"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(timeBuckets.buckets[0]))
	assert.Equal(t, common.HexToHash("1"), timeBuckets.buckets[0][0])
	assert.Equal(t, common.HexToHash("2"), timeBuckets.buckets[0][1])
	assert.Equal(t, 1, len(timeBuckets.buckets[1]))
	assert.Equal(t, common.HexToHash("3"), timeBuckets.buckets[1][0])
	assert.Equal(t, 40, timeBuckets.cap)

	// same bucket and same hash
	err = timeBuckets.Add(BucketDuration*4-1, common.HexToHash("3"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(timeBuckets.buckets[1]))
	assert.Equal(t, common.HexToHash("3"), timeBuckets.buckets[1][0])
	assert.Equal(t, common.HexToHash("3"), timeBuckets.buckets[1][1]) // redundant
	assert.Equal(t, 0, len(timeBuckets.buckets[2]))
	assert.Equal(t, 40, timeBuckets.cap)

	// extend memory
	oldCap := timeBuckets.cap
	err = timeBuckets.Add(BucketDuration*uint32(oldCap)+timeBuckets.TimeBase, common.HexToHash("4"))
	assert.NoError(t, err)
	assert.Equal(t, oldCap*2, timeBuckets.cap)
	assert.Equal(t, 0, len(timeBuckets.buckets[2]))
	assert.Equal(t, 1, len(timeBuckets.buckets[oldCap]))
	assert.Equal(t, common.HexToHash("4"), timeBuckets.buckets[oldCap][0])

	// extend memory 2
	oldCap = timeBuckets.cap
	expectIndex := oldCap*2 + 10
	err = timeBuckets.Add(BucketDuration*uint32(expectIndex)+timeBuckets.TimeBase, common.HexToHash("5"))
	assert.NoError(t, err)
	assert.Equal(t, expectIndex*2, timeBuckets.cap)
	assert.Equal(t, 1, len(timeBuckets.buckets[expectIndex]))
	assert.Equal(t, common.HexToHash("5"), timeBuckets.buckets[expectIndex][0])

	// extend after Expire
	timeBuckets = newTimeBucket(0)
	oldCap = timeBuckets.cap
	_ = timeBuckets.Expire(BucketDuration * uint32(timeBuckets.cap-1))
	assert.Equal(t, 1, len(timeBuckets.buckets))
	err = timeBuckets.Add(BucketDuration*uint32(timeBuckets.cap), common.HexToHash("5"))
	assert.NoError(t, err)
	assert.Equal(t, oldCap, timeBuckets.cap)
}

func TestTimeBuckets_Expire(t *testing.T) {
	// no bucket
	timeBuckets := newTimeBucket(BucketDuration * 2)
	hashList := timeBuckets.Expire(BucketDuration*2 - 1)
	assert.Equal(t, 0, len(hashList))
	assert.Equal(t, BucketDuration*2, timeBuckets.TimeBase)
	hashList = timeBuckets.Expire(BucketDuration*2 + 1)
	assert.Equal(t, 0, len(hashList))
	assert.Equal(t, BucketDuration*2, timeBuckets.TimeBase)
	hashList = timeBuckets.Expire(BucketDuration * 3)
	assert.Equal(t, 0, len(hashList))
	assert.Equal(t, BucketDuration*3, timeBuckets.TimeBase)

	// one bucket
	timeBuckets = newTimeBucket(BucketDuration * 2)
	err := timeBuckets.Add(BucketDuration*2, common.HexToHash("11"))
	assert.NoError(t, err)
	hashList = timeBuckets.Expire(BucketDuration*2 - 1)
	assert.Equal(t, 0, len(hashList))
	assert.Equal(t, BucketDuration*2, timeBuckets.TimeBase)
	hashList = timeBuckets.Expire(BucketDuration*2 + 1)
	assert.Equal(t, 0, len(hashList))
	assert.Equal(t, BucketDuration*2, timeBuckets.TimeBase)
	hashList = timeBuckets.Expire(BucketDuration * 3)
	assert.Equal(t, 1, len(hashList))
	assert.Equal(t, common.HexToHash("11"), hashList[0])
	assert.Equal(t, 0, len(timeBuckets.buckets[0]))
	assert.Equal(t, BucketDuration*3, timeBuckets.TimeBase)

	// expire 1 bucket and 2 blocks in it
	timeBuckets = newTimeBucket(BucketDuration * 2)
	_ = timeBuckets.Add(BucketDuration*2, common.HexToHash("11"))
	_ = timeBuckets.Add(BucketDuration*2+1, common.HexToHash("12"))
	_ = timeBuckets.Add(BucketDuration*3, common.HexToHash("21"))
	hashList = timeBuckets.Expire(BucketDuration * 3)
	assert.Equal(t, 2, len(hashList))
	assert.Equal(t, common.HexToHash("11"), hashList[0])
	assert.Equal(t, common.HexToHash("12"), hashList[1])
	assert.Equal(t, 1, len(timeBuckets.buckets[0]))
	assert.Equal(t, common.HexToHash("21"), timeBuckets.buckets[0][0])
	assert.Equal(t, 0, len(timeBuckets.buckets[1]))
	assert.Equal(t, BucketDuration*3, timeBuckets.TimeBase)

	// expire 2 bucket and 2 blocks in every bucket
	timeBuckets = newTimeBucket(BucketDuration * 2)
	_ = timeBuckets.Add(BucketDuration*2, common.HexToHash("11"))
	_ = timeBuckets.Add(BucketDuration*2+1, common.HexToHash("12"))
	_ = timeBuckets.Add(BucketDuration*3, common.HexToHash("21"))
	_ = timeBuckets.Add(BucketDuration*4-1, common.HexToHash("22"))
	hashList = timeBuckets.Expire(BucketDuration * 4)
	assert.Equal(t, 4, len(hashList))
	assert.Equal(t, common.HexToHash("11"), hashList[0])
	assert.Equal(t, common.HexToHash("12"), hashList[1])
	assert.Equal(t, common.HexToHash("21"), hashList[2])
	assert.Equal(t, common.HexToHash("22"), hashList[3])
	assert.Equal(t, 0, len(timeBuckets.buckets[0]))
	assert.Equal(t, BucketDuration*4, timeBuckets.TimeBase)

	// expire all buckets by a big time
	timeBuckets = newTimeBucket(BucketDuration * 2)
	_ = timeBuckets.Add(BucketDuration*2, common.HexToHash("11"))
	_ = timeBuckets.Add(BucketDuration*3, common.HexToHash("21"))
	hashList = timeBuckets.Expire(BucketDuration * 10)
	assert.Equal(t, 2, len(hashList))
	assert.Equal(t, common.HexToHash("11"), hashList[0])
	assert.Equal(t, common.HexToHash("21"), hashList[1])
	assert.Equal(t, 0, len(timeBuckets.buckets[0]))
	assert.Equal(t, BucketDuration*10, timeBuckets.TimeBase)
}
