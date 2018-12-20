lemo._createAPI('account', [
    {name: 'newKeyPair', method: 'account_newKeyPair'},
]);
lemo._createAPI('mine', [
    {name: 'start', method: 'mine_mineStart'},
    {name: 'stop', method: 'mine_mineStop'},
]);
lemo._createAPI('net', [
    {name: 'connect', method: 'net_connect'},
    {name: 'disconnect', method: 'net_disconnect'},
    {name: 'getConnections', method: 'net_connections'},
]);
lemo._createAPI('tx', [
    {name: 'readContract', method: 'tx_readContract'},
    {name: 'estimateGas', method: 'tx_estimateGas'},
    {name: 'estimateCreateContractGas', method: 'tx_estimateCreateContractGas'},
]);
