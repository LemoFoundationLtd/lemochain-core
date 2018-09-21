package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/base26"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
)

// 地址前的"Lemo" logo标志号
const logo = "Lemo"

type Wallet struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

// newWalletFromECDSA 通过私钥返回地址钱包
func newWalletFromECDSA(privateKey *ecdsa.PrivateKey) *Wallet {
	w := &Wallet{
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey: privateKey,
	}
	return w
}

// newWallet 生成钱包
func NewWallet() (*Wallet, error) {
	// 通过椭圆曲线算法随机生成私钥
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return newWalletFromECDSA(privateKey), nil
}

// getCheckSum 做异或运算得到校验位
func (w *Wallet) getCheckSum() byte {
	address := w.Address.Bytes()
	var temp = address[0]
	for _, c := range address {
		temp ^= c
	}
	return temp
}

// 得到Lemo地址
func (w *Wallet) GenerateAddress() string {
	// 得到校验位
	checkSum := w.getCheckSum()
	// 拼接校验位到最后
	fullPayload := append(w.Address.Bytes(), checkSum)
	// base26编码
	address := base26.Encode(fullPayload)
	// 开头添加logo
	LemoAddress := append([]byte(logo), address...)

	return string(LemoAddress)
}

// ValidateAddress 校验用户输入的LemoAddress是否正确,并返回Address
func ValidateAddress(LemoAddress string) (bool, common.Address) {
	// 移除logo
	Address := []byte(LemoAddress)[4:]
	// Base26解码
	fullPayload := base26.Decode(Address)
	// 得到原生地址ethAdd和校验位actualCheck
	actualCheck := fullPayload[len(fullPayload)-1]
	address := fullPayload[:len(fullPayload)-1]
	w := &Wallet{}
	w.Address = common.BytesToAddress(address)
	targetCheck := w.getCheckSum()
	// 判断目标检验位和拆分出来的校验位是否相等
	if actualCheck == targetCheck {
		return true, w.Address
	}
	return false, w.Address
}
