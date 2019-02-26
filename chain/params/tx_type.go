package params

const (
	OrdinaryTx uint8 = 0 // 普通交易,包括转账交易、创建智能合约交易、调用智能合约交易
	VoteTx     uint8 = 1 // 用户发送投票交易
	RegisterTx uint8 = 2 // 申请参加竞选节点投票交易

	IssueTokenTx            uint8 = 3 // 发行token交易
	AdditionalTokenTx       uint8 = 4 // 增发token交易
	TradingTokenTx          uint8 = 5 // 交易token交易
	IssueAssertTx           uint8 = 6 // 发行资产交易
	AdditionalAssertTx      uint8 = 7 // 增发资产交易
	TradingAssertTx         uint8 = 8 // 交易资产交易
	ModifyTokenAssertInfoTx uint8 = 9 // 修改token/资产的info交易

)
