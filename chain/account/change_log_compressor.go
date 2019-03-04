package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
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
		(logType == StorageLog) ||
		(logType == StorageRootLog) ||
		(logType == AssetCodeLog) ||
		(logType == AssetCodeRootLog) ||
		(logType == AssetIdLog) ||
		(logType == AssetIdRootLog) ||
		(logType == EquityLog) ||
		(logType == EquityRootLog) ||
		(logType == CandidateLog) {
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
		return true
	} else {
		return false
	}
}

// merge traverses change logs and merges change log into the same type one which in front of it
func merge(logs types.ChangeLogSlice) types.ChangeLogSlice {
	result := make(types.ChangeLogSlice, 0)

	combineResult := make([][]*types.ChangeLog, LOG_TYPE_STOP)
	for _, log := range logs {
		if needMerge(log.LogType) {
			if needDel(log.LogType) {
				continue
			}

			if combineResult[log.LogType] == nil {
				combineResult[log.LogType] = make([]*types.ChangeLog, 1)
				combineResult[log.LogType][0] = log
			} else {
				combineResult[log.LogType][0].NewVal = log.NewVal
				combineResult[log.LogType][0].Extra = log.Extra
			}
		} else {
			if combineResult[log.LogType] == nil {
				combineResult[log.LogType] = make([]*types.ChangeLog, 0)
			}
			combineResult[log.LogType] = append(combineResult[log.LogType], log.Copy())
		}
	}

	for i := 0; i < len(combineResult); i++ {
		for j := 0; j < len(combineResult[i]); j++ {
			result = append(result, combineResult[i][j].Copy())
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
