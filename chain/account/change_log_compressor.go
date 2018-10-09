package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"sort"
)

// MergeChangeLogs merges the change logs for same account in block. Then return the merged change logs and changed account versions.
func MergeChangeLogs(logs types.ChangeLogSlice) (types.ChangeLogSlice, map[common.Address]uint32) {
	changedVersions := make(map[common.Address]uint32)
	logsByAccount := make(map[common.Address]types.ChangeLogSlice)
	// classify
	for _, log := range logs {
		logsByAccount[log.Address] = append(logsByAccount[log.Address], log)
	}
	// merge logs in account
	for addr, accountLogs := range logsByAccount {
		newAccountLogs := merge(accountLogs)
		changed, lastVersion := resetVersion(newAccountLogs)
		if changed {
			changedVersions[addr] = lastVersion
		}
		logsByAccount[addr] = newAccountLogs
	}
	// sort all logs by account
	accounts := make(common.AddressSlice, 0, len(logsByAccount))
	for addr := range logsByAccount {
		accounts = append(accounts, addr)
	}
	sort.Sort(accounts)
	mergedLogs := make(types.ChangeLogSlice, 0)
	for _, addr := range accounts {
		mergedLogs = append(mergedLogs, logsByAccount[addr]...)
	}
	return mergedLogs, changedVersions
}

// merge traverses change logs and merges change log into the same type one which in front of it
func merge(logs types.ChangeLogSlice) types.ChangeLogSlice {
	result := logs[:]
	for _, log := range logs {
		exist := result.FindByType(log)
		if log.LogType == BalanceLog || log.LogType == StorageLog {
			// update the exist one
			exist.NewVal = log.NewVal
			exist.Extra = log.Extra
		} else {
			result = append(result, log)
		}
	}
	return logs
}

// resetVersion reset change logs version, then return the last change log as account's version
func resetVersion(logs types.ChangeLogSlice) (bool, uint32) {
	if len(logs) == 0 {
		return false, 0
	}
	oldVersion := logs[len(logs)-1].Version
	for i := 1; i < len(logs); i++ {
		logs[i].Version = logs[i-1].Version + 1
	}
	newVersion := logs[len(logs)-1].Version
	return oldVersion != newVersion, newVersion
}
