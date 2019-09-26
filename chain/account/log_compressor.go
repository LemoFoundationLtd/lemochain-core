package account

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"sort"
)

// MergeChangeLogs merges the change logs for same account in block. Then return the merged change logs.
func MergeChangeLogs(logs types.ChangeLogSlice) types.ChangeLogSlice {
	logsByAccount := make(map[common.Address]types.ChangeLogSlice)
	// classify
	for _, log := range logs {
		logsByAccount[log.Address] = append(logsByAccount[log.Address], log)
	}
	// merge logs in account
	for addr, accountLogs := range logsByAccount {
		newAccountLogs := merge(accountLogs)
		newAccountLogs = removeUnchanged(newAccountLogs)
		// resetVersion(newAccountLogs)
		logsByAccount[addr] = newAccountLogs
	}
	// sort all logs by account
	addressList := make(common.AddressSlice, 0, len(logsByAccount))
	for addr := range logsByAccount {
		addressList = append(addressList, addr)
	}
	sort.Sort(addressList)
	mergedLogs := make(types.ChangeLogSlice, 0)
	for _, addr := range addressList {
		mergedLogs = append(mergedLogs, logsByAccount[addr]...)
	}
	return mergedLogs
}

func needMerge(logType types.ChangeLogType) bool {
	if (logType == BalanceLog) ||
		(logType == VoteForLog) ||
		(logType == VotesLog) ||
		(logType == StorageRootLog) ||
		(logType == AssetCodeLog) ||
		(logType == AssetCodeRootLog) ||
		(logType == AssetCodeTotalSupplyLog) ||
		(logType == AssetIdRootLog) ||
		(logType == EquityLog) ||
		(logType == EquityRootLog) {
		return true
	} else {
		return false
	}
}

func needDel(logType types.ChangeLogType) bool {
	if (logType == StorageLog) ||
		(logType == AssetCodeLog) ||
		(logType == AssetIdLog) ||
		(logType == EquityLog) {
		return false
	} else {
		return false
	}
}

// merge traverses change logs and merges change log into the same type one which in front of it
func merge(logs types.ChangeLogSlice) types.ChangeLogSlice {
	// 缓存数组中的需要merge的数据和下标的索引
	indexMap := make(map[types.ChangeLogType]int)
	// 缓存merge后的changelog
	result := make(types.ChangeLogSlice, 0)
	for _, log := range logs {
		// 判断是否需要merge,如果不需要merge则直接保存到结果数组中
		if needMerge(log.LogType) {
			// 通过map查找数组中是否已经存放了此类log
			if i, ok := indexMap[log.LogType]; ok {
				// 存在则进行merge
				result[i].NewVal = log.NewVal
				result[i].Extra = log.Extra
			} else {
				// 不存在则表示第一次出现,push到数组并建立索引
				result = append(result, log.Copy())
				// 缓存数组index索引
				indexMap[log.LogType] = len(result) - 1
			}
		} else {
			// 不需要merge的changelog就直接按照顺序push到数组中
			result = append(result, log.Copy())
		}
	}
	return result
}

// removeUnchanged removes the unchanged log
func removeUnchanged(logs types.ChangeLogSlice) types.ChangeLogSlice {
	result := make(types.ChangeLogSlice, 0)
	for _, log := range logs {
		if IsValuable(log) {
			result = append(result, log)
		}
	}
	return result
}
