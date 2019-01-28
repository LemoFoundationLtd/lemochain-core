lemo._createAPI('account', [
    {name: 'newKeyPair', method: 'account_newKeyPair'},
    {name: 'getVoteFor', method: 'account_getVoteFor'},
    {name: 'getCandidateInfo', method: 'account_getCandidateInfo'},

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
    {name: 'getTxListByAddress', method: 'tx_getTxListByAddress'},
    {name: 'getTxByHash',method: 'tx_getTxByHash'},
]);
