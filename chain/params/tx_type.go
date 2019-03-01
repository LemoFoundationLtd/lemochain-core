package params

const (
	OrdinaryTx       uint8 = 0 // 普通交易,包括转账交易、创建智能合约交易、调用智能合约交易
	VoteTx           uint8 = 1 // 用户发送投票交易
	RegisterTx       uint8 = 2 // 申请参加竞选节点投票交易
	CreateAssetTx    uint8 = 3 // 创建资产
	IssueAssetTx     uint8 = 4 // 发行资产
	ReplenishAssetTx uint8 = 5 // 增发资产交易
	ModifyAssetTx    uint8 = 6 // 修改资产交易
	TradingAssetTx   uint8 = 7 // 交易资产

)
