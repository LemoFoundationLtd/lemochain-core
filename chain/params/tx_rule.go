package params

const (
	// Max bytes length of extra data in block header
	MaxExtraDataLen = 256
	// Max acceptable transaction's life time. For control tx pool size.
	// Transactions will stay in tx pool till they expired or be packaged
	MaxTxLifeTime = 30 * 60
)
