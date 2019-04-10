lemo._createAPI('account', 'getAllRewardValue', 'account_getAllRewardValue');

lemo._createAPI('mine', 'start', 'mine_mineStart');
lemo._createAPI('mine', 'stop', 'mine_mineStop');

lemo._createAPI('net', 'connect', 'net_connect');
lemo._createAPI('net', 'disconnect', 'net_disconnect');
lemo._createAPI('net', 'getConnections', 'net_connections');

lemo._createAPI('tx', 'estimateGas', 'tx_estimateGas');
lemo._createAPI('tx', 'estimateCreateContractGas', 'tx_estimateCreateContractGas');
lemo._createAPI('tx', 'sendReimbursedGasTx', 'tx_sendReimbursedGasTx');
lemo._createAPI('tx', 'createAsset', 'tx_createAsset');
lemo._createAPI('tx', 'issueAsset', 'tx_issueAsset');
lemo._createAPI('tx', 'replenishAsset', 'tx_replenishAsset');
lemo._createAPI('tx', 'modifyAsset', 'tx_modifyAsset');
lemo._createAPI('tx', 'tradingAsset', 'tx_tradingAsset');

lemo._createAPI('chain', 'getAllDeputyNodesList', 'chain_getAllDeputyNodesList');


function getNewestUnstableBlock() {
    var parseBlock = this.parser.parseBlock;
    var chainID = this.chainID;
    return this.requester.send('chain_unstableBlock', [true])
        .then(function (block) {
            return parseBlock(chainID, block, true)
        })
}
lemo._createAPI('', 'getNewestUnstableBlock', getNewestUnstableBlock);
lemo._createAPI('', 'getNewestUnstableHeight', 'chain_unstableHeight');