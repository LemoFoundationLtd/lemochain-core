package transaction

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"sort"
)

var (
	ErrAddressRepeat = errors.New("cannot set two identical addresses")
	ErrWeight        = errors.New("signer weight error")
	ErrTempAccount   = errors.New("temp account format error")
	ErrSignersNumber = errors.New("cannot exceed the maximum number of signers")
)

type SetMultisigAccountEnv struct {
	am *account.Manager
}

func NewSetMultisigAccountEnv(am *account.Manager) *SetMultisigAccountEnv {
	return &SetMultisigAccountEnv{
		am: am,
	}
}

// unmarshalAndVerifyData
func unmarshalAndVerifyData(data []byte) (types.Signers, error) {
	newSigners := make(types.Signers, 0)
	err := json.Unmarshal(data, &newSigners)
	if err != nil {
		return nil, err
	}

	if len(newSigners) > MaxNumberSigners {
		log.Errorf("Cannot exceed the maximum number of signers. signers number: %d,MaxNumberSigners: %d", len(newSigners), MaxNumberSigners)
		return nil, ErrSignersNumber
	}

	m := make(map[common.Address]uint8)
	for _, v := range newSigners {
		// 验证每一个weight的取值范围
		if v.Weight < 1 || v.Weight > SignersWeight {
			log.Errorf("Weight should be in range [1, 100]. signerAddress: %s, weight: %d", v.Address.String(), v.Weight)
			return nil, ErrWeight
		}
		// 验证不能有相同的地址
		if _, ok := m[v.Address]; ok {
			return nil, ErrAddressRepeat
		}
		m[v.Address] = v.Weight
	}

	return newSigners, nil
}

// judgeTotalWeight
func judgeTotalWeight(signers types.Signers) error {
	var totalWeight int64 = 0
	for _, v := range signers {
		totalWeight = totalWeight + int64(v.Weight)
	}
	if totalWeight < SignersWeight {
		return ErrTotalWeight
	}
	return nil
}

// setMultisigAccount 设置多签账户
func setMultisigAccount(signers types.Signers, toAcc types.AccountAccessor) error {

	if len(signers) > MaxNumberSigners {
		log.Errorf("Cannot exceed the maximum number of signers. signers number: %d,MaxNumberSigners: %d", len(signers), MaxNumberSigners)
		return ErrSignersNumber
	}
	// 验证多签账户的总的weight必须大于100
	err := judgeTotalWeight(signers)
	if err != nil {
		return err
	}
	// 按照字典序排序
	sort.Sort(signers)
	// 设置
	err = toAcc.SetSingers(signers)
	if err != nil {
		return err
	}
	return nil
}

// modifyMultisigAccount 修改多重签名账户中的签名者列表
func modifyMultisigAccount(modifySigners, oldSigners types.Signers, toAcc types.AccountAccessor) (err error) {
	// 把交易传入的签名者列表放入map中
	tempMap := make(map[common.Address]uint8)
	for _, v := range modifySigners {
		tempMap[v.Address] = v.Weight
	}
	// 修改已存在的signer,并从map中删除已存在的signer
	for i := 0; i < len(oldSigners); i++ {
		if _, ok := tempMap[oldSigners[i].Address]; ok {
			oldSigners[i].Weight = tempMap[oldSigners[i].Address]
			delete(tempMap, oldSigners[i].Address)
		}
	}

	newSigners := oldSigners
	// 遍历剩下map中新加的signers
	if len(tempMap) != 0 {
		endSignerAcc := make(types.Signers, 0)
		for k, v := range tempMap {
			endSignerAcc = append(endSignerAcc, types.SignAccount{
				Address: k,
				Weight:  v,
			})
		}
		newSigners = append(newSigners, endSignerAcc...)
	}

	err = setMultisigAccount(newSigners, toAcc)
	return err
}

// ModifyMultisigTx
func (s *SetMultisigAccountEnv) ModifyMultisigTx(from, to common.Address, data []byte) error {
	txSigners, err := unmarshalAndVerifyData(data)
	if err != nil {
		return err
	}

	if from == to { // 普通账户
		toAcc := s.am.GetAccount(to)
		oldSigners := toAcc.GetSigners()

		// 1. 创建多签账户
		if len(oldSigners) == 0 {
			err = setMultisigAccount(txSigners, toAcc)
			if err != nil {
				return err
			}
		} else { // 2. 修改签名者信息
			err = modifyMultisigAccount(txSigners, oldSigners, toAcc)
			if err != nil {
				return err
			}
		}
	} else { // 临时账户
		// 验证临时账户to
		if bytes.Compare(to[1:10], from[11:20]) != 0 {
			return ErrTempAccount
		}
		// todo 临时账户逻辑
		return nil
	}
	return nil
}
