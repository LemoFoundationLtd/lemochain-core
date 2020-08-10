lemo._createAPI('account', 'getAssetEquityByAssetId', 'account_getAssetEquityByAssetId');

lemo._createAPI('mine', 'start', 'mine_mineStart');
lemo._createAPI('mine', 'stop', 'mine_mineStop');
lemo._createAPI('mine', 'setLeastGasPrice', 'mine_setLeastGasPrice');
lemo._createAPI('mine', 'getLeastGasPrice', 'mine_getLeastGasPrice');

lemo._createAPI('net', 'connect', 'net_connect');
lemo._createAPI('net', 'disconnect', 'net_disconnect');
lemo._createAPI('net', 'getConnections', 'net_connections');
lemo._createAPI('net', 'broadcastConfirm', 'net_broadcastConfirm');
lemo._createAPI('net', 'fetchConfirm', 'net_fetchConfirm');

lemo._createAPI('tx', 'getPendingTx', 'tx_getPendingTx');

lemo._createAPI('chain', 'getAllRewardValue', 'chain_getAllRewardValue');
lemo._createAPI('chain', 'logForks', 'chain_logForks');
