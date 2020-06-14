package params

const (
	// Max bytes length of extra data in block header
	MaxExtraDataLen = 256
	MaxTxLifeTime   = 30 * 60 // 为控制交易池尺寸，只接收寿命为30分钟以内的交易
)
