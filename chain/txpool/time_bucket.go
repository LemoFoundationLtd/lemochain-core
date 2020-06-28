package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// TimeBuckets中每一个slot覆盖的时间范围，单位为秒
const BucketDuration = uint32(60)

type HashList []common.Hash
type TimeBuckets struct {
	// 第一个bucket的起始时间，是BucketDuration的整数倍。此时间戳是随着链上的stable变化而变化的
	TimeBase uint32

	/** 从TimeBase开始，按BucketDuration划分bucket，保存hash。它的空间是预分配好的
	 * 任何一个Hash所在的bucket下标为： 过期时间/BucketDuration - TimeBase/BucketDuration
	 * 第i个桶的时间范围为[TimeBase + i*BucketDuration, TimeBase + (i+1)*BucketDuration)
	 */
	buckets []HashList
	cap     int
}

func newTimeBucket(timeBase uint32) *TimeBuckets {
	timeBucket := &TimeBuckets{}
	timeBucket.TimeBase = uint32(timeBase/BucketDuration) * BucketDuration
	timeBucket.cap = params.MaxTxLifeTime/int(BucketDuration) + 10
	timeBucket.buckets = make([]HashList, timeBucket.cap)
	return timeBucket
}

// getBucketIndex calculate bucket index by time
func (timeBucket *TimeBuckets) getBucketIndex(time uint32) int {
	return int(time/BucketDuration) - int(timeBucket.TimeBase/BucketDuration)
}

// Add save hash into time bucket
func (timeBucket *TimeBuckets) Add(time uint32, hash common.Hash) error {
	index := timeBucket.getBucketIndex(time)
	if index < 0 {
		return ErrTimeBucketTime
	}

	// extend storage
	if index >= len(timeBucket.buckets) {
		if index >= timeBucket.cap {
			timeBucket.cap = index * 2
		}
		tmp := make([]HashList, timeBucket.cap)
		copy(tmp, timeBucket.buckets)
		timeBucket.buckets = tmp
	}
	if timeBucket.buckets[index] == nil {
		timeBucket.buckets[index] = make(HashList, 0, 1)
	}
	// redundancy is fine
	timeBucket.buckets[index] = append(timeBucket.buckets[index], hash)

	return nil
}

// Expire delete all expired data by move time base pointer
func (timeBucket *TimeBuckets) Expire(newTimeBase uint32) []common.Hash {
	result := make([]common.Hash, 0)
	// the new first bucket position
	newBaseIndex := timeBucket.getBucketIndex(newTimeBase)
	if newBaseIndex <= 0 {
		return result
	}
	// it means all data will be deleted
	if newBaseIndex > len(timeBucket.buckets) {
		newBaseIndex = len(timeBucket.buckets)
	}
	// collect deleted hash list
	for i := 0; i < newBaseIndex; i++ {
		hashes := timeBucket.buckets[i]
		result = append(result, hashes...)
	}

	// update buckets
	timeBucket.TimeBase = uint32(newTimeBase/BucketDuration) * BucketDuration
	timeBucket.buckets = timeBucket.buckets[newBaseIndex:]

	return result
}
