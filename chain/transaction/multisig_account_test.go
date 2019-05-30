package transaction

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

// newSignersData 创建签名者列表的数据
type testMap map[uint8]common.Address

func newSignersData(mm testMap) []byte {
	signers := make(types.Signers, 0)
	for k, v := range mm {
		signerAcc := types.SignAccount{
			Address: v,
			Weight:  k,
		}
		signers = append(signers, signerAcc)
	}

	data, err := json.Marshal(signers)
	if err != nil {
		panic(err)
	}
	return data
}

// Test_unmarshalAndVerifyData
func Test_unmarshalAndVerifyData(t *testing.T) {
	// 1. 验证weight小于1的情况
	m1 := make(testMap)
	m1[0] = common.HexToAddress("0x11")
	data1 := newSignersData(m1)
	signers1, err := unmarshalAndVerifyData(data1)
	assert.Nil(t, signers1)
	assert.Equal(t, ErrWeight, err)

	// 2. 验证有相同地址的情况
	m2 := make(testMap)
	m2[2] = common.HexToAddress("0x11")
	m2[4] = common.HexToAddress("0x12")
	m2[3] = common.HexToAddress("0x11")
	data2 := newSignersData(m2)
	signers2, err := unmarshalAndVerifyData(data2)
	assert.Nil(t, signers2)
	assert.Equal(t, ErrAddressRepeat, err)

	// 3. 正常情况
	m3 := make(testMap)
	for i := 1; i <= SignerWeightThreshold; i++ {
		m3[uint8(i)] = common.HexToAddress(strconv.Itoa(i))
	}
	data3 := newSignersData(m3)
	signers3, err := unmarshalAndVerifyData(data3)
	assert.NoError(t, err)
	assert.Equal(t, len(m3), len(signers3))

	for _, v := range signers3 {
		assert.Equal(t, v.Address, m3[v.Weight])
	}
}

// Test_judgeTotalWeight
func Test_judgeTotalWeight(t *testing.T) {
	// 总weight数不足100
	m := make(testMap)
	m[55] = common.HexToAddress("0x11")
	m[40] = common.HexToAddress("0x1122")
	data := newSignersData(m)
	signers, err := unmarshalAndVerifyData(data)
	assert.NoError(t, err)
	err = judgeTotalWeight(signers)
	assert.Equal(t, ErrTotalWeight, err)
}

// Test_setMultisigAccount
func Test_setMultisigAccount(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	toAcc := am.GetAccount(common.HexToAddress("0x999"))
	// 测流程
	m := make(testMap)
	m[50] = common.HexToAddress("0x11")
	m[60] = common.HexToAddress("0x1122")
	data := newSignersData(m)
	signers, err := unmarshalAndVerifyData(data)
	assert.NoError(t, err)
	err = setMultisigAccount(signers, toAcc)
	assert.NoError(t, err)
	// 验证结果
	getSigners := toAcc.GetSigners()
	assert.Equal(t, signers, getSigners)
}

// TestSetMultisigAccountEnv_ModifyMultisigTx
func TestSetMultisigAccountEnv_ModifyMultisigTx(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	muEnv := NewSetMultisigAccountEnv(am)
	// 测流程
	from := common.HexToAddress("0x112")
	var to common.Address

	for i := 0; i < 2; i++ {
		if i == 0 {
			to = from // 普通账户
		} else {
			to = crypto.CreateTempAddress(from, [10]byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9}) // 临时账户
		}

		signer01 := common.HexToAddress("0x111")
		signer02 := common.HexToAddress("0x112")
		signer03 := common.HexToAddress("0x113")
		m1 := make(testMap)
		m1[50] = signer01
		m1[60] = signer02
		m1[70] = signer03
		data01 := newSignersData(m1)

		// 1. 创建多重签名账户测试
		err := muEnv.ModifyMultisigTx(from, to, data01)
		assert.NoError(t, err)
		toAcc := am.GetAccount(to)
		mm := make(testMap)
		signers01 := toAcc.GetSigners()
		for _, v := range signers01 {
			mm[v.Weight] = v.Address
		}
		assert.Equal(t, mm[50], m1[50])
		assert.Equal(t, mm[60], m1[60])
		assert.Equal(t, mm[70], m1[70])

		// 2. 修改多重签名账户测试
		m2 := make(testMap)
		// 所有权重减10
		m2[40] = signer01
		m2[50] = signer02
		m2[60] = signer03
		data02 := newSignersData(m2)
		err = muEnv.ModifyMultisigTx(to, to, data02)
		assert.NoError(t, err)

		signers02 := toAcc.GetSigners()
		mmm := make(testMap)
		for _, v := range signers02 {
			mmm[v.Weight] = v.Address
		}
		assert.Equal(t, mmm[40], m1[50])
		assert.Equal(t, mmm[50], m1[60])
		assert.Equal(t, mmm[60], m1[70])
	}

}

// Test_verifyTempAddress
func Test_verifyTempAddress(t *testing.T) {
	versionErrTempAddr := common.Address{99, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 8, 7, 5, 3, 4, 3, 3, 3, 3}

	creator := common.Address{common.LemoAddressVersion, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	fieldErrTempAddr := common.Address{common.TempAddressVersion, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 8, 7, 5, 3, 4, 3, 3, 3, 3}
	trueTempAddr := crypto.CreateTempAddress(creator, [10]byte{8, 8, 8, 8, 8, 8, 8, 8, 8, 8})

	err := verifyTempAddress(creator, versionErrTempAddr)
	assert.Equal(t, ErrAddressVersion, err)

	err = verifyTempAddress(creator, fieldErrTempAddr)
	assert.Equal(t, ErrTempAddress, err)

	err = verifyTempAddress(creator, trueTempAddr)
	assert.NoError(t, err)
}
