lemo._createAPI('account', 'getAllRewardValue', 'account_getAllRewardValue');
lemo._createAPI('account', 'getAssetEquityByAssetId', 'account_getAssetEquityByAssetId');

lemo._createAPI('mine', 'start', 'mine_mineStart');
lemo._createAPI('mine', 'stop', 'mine_mineStop');

lemo._createAPI('net', 'connect', 'net_connect');
lemo._createAPI('net', 'disconnect', 'net_disconnect');
lemo._createAPI('net', 'getConnections', 'net_connections');

lemo._createAPI('tx', 'estimateGas', 'tx_estimateGas');
lemo._createAPI('tx', 'estimateCreateContractGas', 'tx_estimateCreateContractGas');

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