package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"sort"
)

// MergeChangeLogs merges the change logs for same account in block. Then update the version of change logs and account.
func MergeChangeLogs(logs types.ChangeLogSlice, am *Manager) types.ChangeLogSlice {
	logsByAccount := make(map[common.Address]types.ChangeLogSlice)
	// classify
	for _, log := range logs {
		logsByAccount[log.Address] = append(logsByAccount[log.Address], log)
	}
	// merge logs in account
	for addr, accountLogs := range logsByAccount {
		newAccountLogs := merge(accountLogs)
		resetVersion(newAccountLogs, am)
		logsByAccount[addr] = newAccountLogs
	}
	// sort all logs by account
	accounts := make(common.AddressSlice, 0, len(logsByAccount))
	for addr := range logsByAccount {
		accounts = append(accounts, addr)
	}
	sort.Sort(accounts)
	result := make(types.ChangeLogSlice, 0)
	for _, addr := range accounts {
		result = append(result, logsByAccount[addr]...)
	}
	return result
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

// resetVersion reset change logs version, then update account's version by the last change log
func resetVersion(logs types.ChangeLogSlice, am *Manager) {
	if len(logs) == 0 {
		return
	}
	for i := 1; i < len(logs); i++ {
		logs[i].Version = logs[i-1].Version + 1
	}
	lastVersion := logs[len(logs)-1].Version
	account := am.getRawAccount(logs[0].Address)
	account.SetVersion(lastVersion)
}
