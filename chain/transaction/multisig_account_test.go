package transaction

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
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
	m2[3] = common.HexToAddress("0x11")
	data2 := newSignersData(m2)
	signers2, err := unmarshalAndVerifyData(data2)
	assert.Nil(t, signers2)
	assert.Equal(t, ErrAddressRepeat, err)

	// 3. 正常情况
	m3 := make(testMap)
	for i := 1; i <= SignersWeight; i++ {
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

// Test_modifyMultisigAccount
func Test_modifyMultisigAccount(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	toAcc := am.GetAccount(common.HexToAddress("0x999"))
	// 签名者
	signer01 := common.HexToAddress("0x111")
	signer02 := common.HexToAddress("0x112")
	signer03 := common.HexToAddress("0x113")
	signer04 := common.HexToAddress("0x114")
	signer05 := common.HexToAddress("0x115")
	signer06 := common.HexToAddress("0x116")
	signer07 := common.HexToAddress("0x117")
	signer08 := common.HexToAddress("0x118")
	// 测流程
	m1 := make(testMap)
	m1[10] = signer01
	m1[20] = signer02
	m1[30] = signer03
	m1[40] = signer04
	m1[50] = signer05
	// 1. 构造一个多重签名账户
	data01 := newSignersData(m1)
	oldSigners, err := unmarshalAndVerifyData(data01)
	assert.NoError(t, err)
	err = toAcc.SetSingers(oldSigners)
	assert.NoError(t, err)

	m2 := make(testMap)
	m2[60] = signer04
	m2[70] = signer05
	m2[80] = signer06
	m2[90] = signer07
	m2[100] = signer08
	// 2. 修改多重签名账户
	data02 := newSignersData(m2)
	modifySigners, err := unmarshalAndVerifyData(data02)
	assert.NoError(t, err)
	err = modifyMultisigAccount(modifySigners, oldSigners, toAcc)
	assert.NoError(t, err)
	newSigners := toAcc.GetSigners()

	// 测试修改结果
	mm := make(testMap)
	for _, v := range newSigners {
		mm[v.Weight] = v.Address
	}
	assert.Equal(t, mm[10], signer01)
	assert.Equal(t, mm[20], signer02)
	assert.Equal(t, mm[30], signer03)
	assert.Equal(t, mm[40], common.Address{})
	assert.Equal(t, mm[50], common.Address{})
	assert.Equal(t, mm[60], signer04)
	assert.Equal(t, mm[70], signer05)
	assert.Equal(t, mm[80], signer06)
	assert.Equal(t, mm[90], signer07)
	assert.Equal(t, mm[100], signer08)
}

// TestSetMultisigAccountEnv_CreateOrModifyMultisigTx
func TestSetMultisigAccountEnv_CreateOrModifyMultisigTx(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	muEnv := NewSetMultisigAccountEnv(am)
	// 测流程
	from := common.HexToAddress("0x112")
	to := from // todo 当测试临时账户的时候from不等于to
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
	err = muEnv.ModifyMultisigTx(from, to, data02)
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
