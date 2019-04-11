package params

// tx type
const (
	OrdinaryTx       uint16 = 0 // 普通交易,包括转账交易、创建智能合约交易、调用智能合约交易
	VoteTx           uint16 = 1 // 用户发送投票交易
	RegisterTx       uint16 = 2 // 申请参加竞选节点投票交易
	CreateAssetTx    uint16 = 3 // 创建资产
	IssueAssetTx     uint16 = 4 // 发行资产
	ReplenishAssetTx uint16 = 5 // 增发资产交易
	ModifyAssetTx    uint16 = 6 // 修改资产交易
	TransferAssetTx  uint16 = 7 // 交易资产

)
