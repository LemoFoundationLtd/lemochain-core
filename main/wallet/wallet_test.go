package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"testing"
)

// TestWallet_GenerateAddress 生成LemoAddress功能测试
func TestWallet_GenerateAddress(t *testing.T) {

	tests := []struct {
		address common.Address
	}{
		{common.BytesToAddress([]byte("728ab306f31031b1e06f"))},
		{common.BytesToAddress([]byte("0x000000000000000001"))},
		{common.BytesToAddress([]byte("0x000000000000000002"))},
		{common.BytesToAddress([]byte("0x000000000000000003"))},
		{common.BytesToAddress([]byte("1e030728ab306f316fb"))},
		{common.BytesToAddress([]byte("6f31031728ab30b1e06f"))},
	}
	for _, test := range tests {
		private, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		w := &Wallet{test.address, private}
		LemoAddress := w.GenerateAddress()
		IsValid, address := ValidateAddress(LemoAddress)
		if bytes.Compare(address.Bytes(), test.address.Bytes()) != 0 {
			t.Errorf("LemoAddress == %v,genaddress(%v) == Validaddress(%v)",
				LemoAddress, string(test.address.Bytes()), string(address.Bytes()))
		}
		t.Logf("LemoAddress = %v\n testAddress = %v,ValidAddress = %v,IsValid = %v\n",
			LemoAddress, string(test.address.Bytes()), string(address.Bytes()), IsValid)
	}
}

// TestValidateAddress 验证用户输入LemoAddress功能测试
func TestValidateAddress(t *testing.T) {
	tests := []struct {
		LemoAddress string
	}{
		{"Lemo4CQG3RWZG8DSQFTY6P78K24RCK88TTRNFSP6"}, // true
		{"Lemo454NG9K7BFDQZR6J93T2QWPAG6Q4N9FWBR7J"}, // true
		{"Lemo454NG9K7BFDQZRQW6J93T2PAG6Q4N9FWBR7J"}, // false
		{"Lemo7BFDQZR6J93T45G9K2QWG6Q4N4N9FWBR7JPA"}, // false
		{"Lemo454NG9K7BFDQZR6J93T2QWPAG6Q4N9FWBRZB"}, // true
	}
	for k, test := range tests {
		switch k {
		case 0, 1, 4: // LemoAddress is true
			IsValid, _ := ValidateAddress(test.LemoAddress)
			if !IsValid {
				t.Fatal("验证函数有误!")
			}
		case 2, 3: // LemoAddress is false
			IsValid, _ := ValidateAddress(test.LemoAddress)
			if IsValid {
				t.Fatal("验证函数有误!")
			}
		}
	}
}

// TestNewWallet 生成钱包功能测试
func TestNewWallet(t *testing.T) {
	for i := 0; i < 5; i++ {
		w, err := NewWallet()
		if err != nil {
			t.Fatal(err)
		}
		// 打印common.Address，验证得到的切片首位为1(版本号0x01),字节为20
		t.Log(w.Address.Bytes())
		// 打印出Lemo链用户得到的地址
		t.Log(w.GenerateAddress())
	}
}
