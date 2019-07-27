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
	ErrAddressRepeat        = errors.New("cannot set two identical addresses")
	ErrWeight               = errors.New("signer weight error")
	ErrTempAccount          = errors.New("temp account format error")
	ErrSignersNumber        = errors.New("cannot exceed the maximum number of signers")
	ErrRepeatSetTempAddress = errors.New("the temp address's multisig has been set up")
)

type SetMultisigAccountEnv struct {
	am *account.Manager
}

func NewSetMultisigAccountEnv(am *account.Manager) *SetMultisigAccountEnv {
	return &SetMultisigAccountEnv{
		am: am,
	}
}

//go:generate gencodec -type ModifySigners -out gen_modify_signers_json.go
type ModifySigners struct {
	Signers types.Signers `json:"signers"  gencodec:"required"`
}

// unmarshalAndVerifyData
func unmarshalAndVerifyData(data []byte) (types.Signers, error) {
	// newSigners := make(types.Signers, 0)
	newSigners := &ModifySigners{}
	err := json.Unmarshal(data, &newSigners)
	if err != nil {
		return nil, err
	}

	if len(newSigners.Signers) > MaxSignersNumber {
		log.Errorf("Cannot exceed the maximum number of signers. signers number: %d,MaxSignersNumber: %d", len(newSigners.Signers), MaxSignersNumber)
		return nil, ErrSignersNumber
	}

	m := make(map[common.Address]uint8)
	for _, v := range newSigners.Signers {
		// 验证每一个weight的取值范围
		if v.Weight < 1 || v.Weight > SignerWeightThreshold {
			log.Errorf("Weight should be in range [1, 100]. signerAddress: %s, weight: %d", v.Address.String(), v.Weight)
			return nil, ErrWeight
		}
		// 验证不能有相同的地址
		if _, ok := m[v.Address]; ok {
			return nil, ErrAddressRepeat
		}
		m[v.Address] = v.Weight
	}

	return newSigners.Signers, nil
}

// judgeTotalWeight
func judgeTotalWeight(signers types.Signers) error {
	var totalWeight int64 = 0
	for _, v := range signers {
		totalWeight = totalWeight + int64(v.Weight)
	}
	if totalWeight < SignerWeightThreshold {
		return ErrTotalWeight
	}
	return nil
}

// setMultisigAccount 设置多签账户
func setMultisigAccount(signers types.Signers, toAcc types.AccountAccessor) error {
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

// ModifyMultisigTx
func (s *SetMultisigAccountEnv) ModifyMultisigTx(from, to common.Address, data []byte) error {
	txSigners, err := unmarshalAndVerifyData(data)
	if err != nil {
		return err
	}
	toAcc := s.am.GetAccount(to)

	if from != to { // 设置临时账户to为多签账户
		// 1. 校验临时账户地址
		err = verifyTempAddress(from, to)
		if err != nil {
			return err
		}
		// 2. 查看临时账户是否已经存在
		signers := toAcc.GetSigners()
		if len(signers) != 0 {
			log.Errorf("The temp address's multisig has been set up. Multiple signers: %s", signers.String())
			return ErrRepeatSetTempAddress
		}
	}

	// 设置多签账户
	err = setMultisigAccount(txSigners, toAcc)
	if err != nil {
		return err
	}
	return nil
}

// verifyTempAddress
func verifyTempAddress(creator, tempAddress common.Address) error {
	// 验证是否为临时地址
	if !tempAddress.IsTempAddress() {
		log.Errorf("Address version wrong. Error version: %d. TempAddress version: %d", tempAddress[0], common.TempAddressType)
		return ErrAddressType
	}

	// 验证 creator[11:] == tempAddress[1:10]
	if bytes.Compare(creator[common.AddressLength-common.TempIssuerBytesLength:], tempAddress[1:1+common.TempIssuerBytesLength]) != 0 {
		log.Errorf("The same bytes error. Same bytes of create address: %d. Temp address: %d", creator[common.AddressLength-common.TempIssuerBytesLength:], tempAddress[1:1+common.TempIssuerBytesLength])
		return ErrTempAddress
	}
	return nil
}
