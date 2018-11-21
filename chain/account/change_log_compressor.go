package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"sort"
)

// MergeChangeLogs merges the change logs for same account in block. Then return the merged change logs and the versions need to be revert.
func MergeChangeLogs(logs types.ChangeLogSlice) (types.ChangeLogSlice, types.ChangeLogSlice) {
	logsByAccount := make(map[common.Address]types.ChangeLogSlice)
	versionRevertLogs := make(types.ChangeLogSlice, 0)
	// classify
	for _, log := range logs {
		logsByAccount[log.Address] = append(logsByAccount[log.Address], log)
	}
	// merge logs in account
	for addr, accountLogs := range logsByAccount {
		newAccountLogs := merge(accountLogs)
		newAccountLogs = removeUnchanged(newAccountLogs)
		resetVersion(newAccountLogs)
		logsByAccount[addr] = newAccountLogs
		if len(newAccountLogs) == 0 {
			versionRevertLogs = append(versionRevertLogs, &types.ChangeLog{
				Address: accountLogs[0].Address,
				LogType: accountLogs[0].LogType,
				Version: accountLogs[0].Version - 1,
			})
		}
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
	return mergedLogs, versionRevertLogs
}

// merge traverses change logs and merges change log into the same type one which in front of it
func merge(logs types.ChangeLogSlice) types.ChangeLogSlice {
	result := make(types.ChangeLogSlice, 0)
	for _, log := range logs {
		exist := result.FindByType(log)
		if exist != nil && (log.LogType == BalanceLog || log.LogType == StorageLog) {
			// update the exist one
			exist.NewVal = log.NewVal
			exist.Extra = log.Extra
		} else {
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

// resetVersion reset change logs version, then return the changed versions
func resetVersion(logs types.ChangeLogSlice) {
	count := len(logs)
	for i := 1; i < count; i++ {
		logs[i].Version = logs[i-1].Version + 1
	}
}
