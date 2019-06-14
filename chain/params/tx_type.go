package params

// tx type
const (
	OrdinaryTx       uint16 = 0  // 普通交易,包括转账交易和调用智能合约交易
	CreateContractTx uint16 = 1  // 创建智能合约交易
	VoteTx           uint16 = 2  // 用户发送投票交易
	RegisterTx       uint16 = 3  // 申请参加竞选节点投票交易
	CreateAssetTx    uint16 = 4  // 创建资产
	IssueAssetTx     uint16 = 5  // 发行资产
	ReplenishAssetTx uint16 = 6  // 增发资产交易
	ModifyAssetTx    uint16 = 7  // 修改资产交易
	TransferAssetTx  uint16 = 8  // 交易资产
	ModifySigsTx     uint16 = 9  // 设置多重签名账户的签名者交易
	BoxTx            uint16 = 10 // 箱子交易

)
