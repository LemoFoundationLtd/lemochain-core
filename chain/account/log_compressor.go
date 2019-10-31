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
		(logType == VotesLog) || // must merge because the candidate ranking logic find changed candidates by logs
		(logType == StorageRootLog) ||
		(logType == AssetCodeRootLog) ||
		(logType == AssetCodeTotalSupplyLog) ||
		(logType == AssetIdRootLog) ||
		(logType == EquityLog) ||
		(logType == EquityRootLog) {
		return true
	} else {
		// AssetCodeLog, CandidateLog, CandidateStateLog are associated with same data in account. If they are merged, the sequence will be changed and the result will be different
		return false
	}
}

// merge traverses change logs and merges change log into the same type one which in front of it
func merge(logs types.ChangeLogSlice) types.ChangeLogSlice {
	// 缓存数组中的需要merge的数据和changeLog的extra与数组下标
	typeMap := make(map[types.ChangeLogType]map[interface{}]int)
	// 缓存merge后的changelog
	result := make(types.ChangeLogSlice, 0)
	for _, log := range logs {
		// 判断是否需要merge,如果不需要merge则直接保存到结果数组中
		if !needMerge(log.LogType) {
			// 不需要merge的changelog就直接按照顺序push到数组中
			result = append(result, log.Copy())
			continue
		}

		// 根据type获取extra到index的映射表
		extraMap, ok := typeMap[log.LogType]
		if !ok {
			extraMap = make(map[interface{}]int)
			typeMap[log.LogType] = extraMap
		}

		// 取出上次出现该extra的index。map支持用nil做key
		if lastLogIndex, ok := extraMap[log.Extra]; ok {
			result[lastLogIndex].NewVal = log.NewVal
		} else {
			// 表示在本类型的changeLog下出现了新的extra，不能被merge，直接保存到数组里
			result = append(result, log.Copy())
			extraMap[log.Extra] = len(result) - 1
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
