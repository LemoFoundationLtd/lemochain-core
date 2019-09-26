package account

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_merge(t *testing.T) {
	/**
	测试思路：
	1. 前面按照顺序10个一组构造了5组可以merge的changeLog，
	2. 接着按照顺序9个一组构造了2组不可以merge的changeLog，
	3. 把构造的这68条changeLog进行merge,测试能merge的changeLog按照顺序merge，不能merge的changeLog按照顺序插入newLogs
	*/
	logs := make(types.ChangeLogSlice, 0)
	// 需要merge类型
	needMergeType := []types.ChangeLogType{BalanceLog, VoteForLog, VotesLog, StorageRootLog, AssetCodeLog, AssetCodeRootLog, AssetCodeTotalSupplyLog, AssetIdRootLog, EquityLog, EquityRootLog}
	// 转换成map形式
	needMap := make(map[types.ChangeLogType]struct{})
	for _, v := range needMergeType {
		needMap[v] = struct{}{}
	}
	// 不需要merge类型
	needlessMergeType := make([]types.ChangeLogType, 0)
	for typ := types.ChangeLogType(1); typ < LOG_TYPE_STOP; typ++ {
		if _, ok := needMap[typ]; !ok {
			needlessMergeType = append(needlessMergeType, typ)
		}
	}
	// 创建一组需要merge的changelog
	for i := 0; i < 5; i++ {
		for i, typ := range needMergeType {
			log := newChangeLog(common.HexToAddress("0x112"), typ, time.Now().String(), i)
			logs = append(logs, log)
		}
	}
	needMergeLogsLength := len(logs)
	t.Log("firstLogsLength", needMergeLogsLength)
	// 追加一组不需要merge的changelog
	for j := 0; j < 2; j++ {
		for i, typ := range needlessMergeType {
			log := newChangeLog(common.HexToAddress("0x112"), typ, time.Now().String(), i)
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
