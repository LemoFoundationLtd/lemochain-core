package account

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Test_merge_noExtra 不存在extra的merge测试
func Test_merge_noExtra(t *testing.T) {
	/**
	测试思路：
	1. 前面按照顺序10个一组构造了5组可以merge的changeLog，
	2. 接着按照顺序9个一组构造了2组不可以merge的changeLog，
	3. 把构造的这68条changeLog进行merge,测试能merge的changeLog按照顺序merge，不能merge的changeLog按照顺序插入newLogs
	*/
	logs := make(types.ChangeLogSlice, 0)
	// 需要merge类型
	needMergeType := make([]types.ChangeLogType, 0)
	// 不需要merge类型
	needlessMergeType := make([]types.ChangeLogType, 0)
	for typ := types.ChangeLogType(1); typ < LOG_TYPE_STOP; typ++ {
		if needMerge(typ) {
			needMergeType = append(needMergeType, typ)
		} else {
			needlessMergeType = append(needlessMergeType, typ)
		}
	}
	// 创建一组需要merge的changelog
	for i := 0; i < 5; i++ {
		for i, typ := range needMergeType {
			log := newChangeLog(common.HexToAddress("0x112"), typ, nil, i)
			logs = append(logs, log)
		}
	}
	needMergeLogsLength := len(logs)
	t.Log("firstLogsLength", needMergeLogsLength)
	// 追加一组不需要merge的changelog
	for j := 0; j < 2; j++ {
		for i, typ := range needlessMergeType {
			log := newChangeLog(common.HexToAddress("0x112"), typ, nil, i)
			logs = append(logs, log)
		}
	}
	needlessMergeLogsLength := len(logs) - needMergeLogsLength
	t.Log("needlessMergeLogsLength", needlessMergeLogsLength)
	newLogs := merge(logs)
	t.Log("newLogs", len(newLogs))
	// 检查Merge之后的changeLog数量
	assert.Equal(t, needMergeLogsLength/5+needlessMergeLogsLength, len(newLogs))
	// 检查merge之后的changelog顺序
	// 1. 需要merge的顺序和个数和needMergeType这个数组一致
	for i := 0; i < len(needMergeType); i++ {
		assert.Equal(t, needMergeType[i], newLogs[i].LogType)
	}
	// 2. newLogs后面的都是不需要merge的changeLog,个数为2 * len(needlessMergeType),并且顺序为needlessMergeType的顺序一致，只是重复了一次这个顺序
	for j := len(needMergeType); j < len(needMergeType)+len(needlessMergeType); j++ { // 第一轮
		assert.Equal(t, needlessMergeType[j-len(needMergeType)], newLogs[j].LogType)
	}
	for k := len(needMergeType) + len(needlessMergeType); k < len(newLogs); k++ { // 第二轮
		assert.Equal(t, needlessMergeType[k-len(needlessMergeType)-len(needMergeType)], newLogs[k].LogType)
	}
}

// Test_merge_extra 存在extra的merge测试
func Test_merge_extra(t *testing.T) {
	logs := make(types.ChangeLogSlice, 0)
	// 创建一组能merge但是存在extra的changeLog
	// 需要merge类型
	needMergeType := make([]types.ChangeLogType, 0)
	for typ := types.ChangeLogType(1); typ < LOG_TYPE_STOP; typ++ {
		if needMerge(typ) {
			needMergeType = append(needMergeType, typ)
		}
	}

	// changeLog类型相同extra相同会merge
	log01 := newChangeLog(common.HexToAddress("0x111"), needMergeType[0], common.HexToHash("0x222"), 111)
	log02 := newChangeLog(common.HexToAddress("0x111"), needMergeType[0], common.HexToHash("0x222"), 222)

	// changeLog类型相同但是extra不相同则不会被merge
	log03 := newChangeLog(common.HexToAddress("0x111"), needMergeType[0], common.HexToHash("0x333"), 333)

	// changeLog类型不同extra相同不会被merge
	log04 := newChangeLog(common.HexToAddress("0x111"), needMergeType[1], common.HexToHash("0x333"), 444)

	// changeLog类型和extra都不相同不会被merge
	log05 := newChangeLog(common.HexToAddress("0x111"), needMergeType[2], common.HexToHash("0x555"), 555)

	logs = append(logs, log01, log02, log03, log04, log05)
	newLogs := merge(logs)

	// merge之后的第一条和log02相等，第二条和log03相等,第三条为log04
	assert.Equal(t, 4, len(newLogs))
	assert.Equal(t, newLogs[0].Extra, common.HexToHash("0x222"))
	assert.Equal(t, newLogs[0].NewVal, 222)

	assert.Equal(t, newLogs[1].Extra, common.HexToHash("0x333"))
	assert.Equal(t, newLogs[1].NewVal, 333)

	assert.Equal(t, newLogs[2].Extra, common.HexToHash("0x333"))
	assert.Equal(t, newLogs[2].NewVal, 444)

	assert.Equal(t, newLogs[3].Extra, common.HexToHash("0x555"))
	assert.Equal(t, newLogs[3].NewVal, 555)

}

func newChangeLog(address common.Address, logType types.ChangeLogType, extra, newVal interface{}) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: logType,
		Address: address,
		Version: 1,
		OldVal:  []byte{},
		NewVal:  newVal,
		Extra:   extra,
	}
}
