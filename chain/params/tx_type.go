package params

const (
	Ordinary_tx uint8 = 0 // 普通交易,包括转账交易、创建智能合约交易、调用智能合约交易
	Vote_tx     uint8 = 1 // 用户发送投票交易
	Register_tx uint8 = 2 // 申请参加竞选节点投票交易
)
