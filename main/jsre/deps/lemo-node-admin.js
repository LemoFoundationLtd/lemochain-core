lemo._createAPI('account', 'newKeyPair', 'newKeyPair');

lemo._createAPI('mine', 'start', 'mineStart');
lemo._createAPI('mine', 'stop', 'mineStop');

lemo._createAPI('net', 'connect', 'connect');
lemo._createAPI('net', 'disconnect', 'disconnect');
lemo._createAPI('net', 'getConnections', 'connections');

// TODO remove
lemo._createAPI('tx', 'readContract', 'readContract');
lemo._createAPI('tx', 'estimateGas', 'estimateGas');
lemo._createAPI('tx', 'estimateCreateContractGas', 'estimateCreateContractGas');
lemo._createAPI('tx', 'getTxListByAddress', 'getTxListByAddress');

lemo._createAPI('chain', 'getCandidateList', 'getCandidateList');