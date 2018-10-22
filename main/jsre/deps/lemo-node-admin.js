lemo.createAPI('account', [
	{name: 'newKeyPair', api: 'account_newKeyPair'},
]);
lemo.createAPI('mine', [
	{name: 'start', api: 'mine_mineStart'},
	{name: 'stop', api: 'mine_mineStop'},
]);
lemo.createAPI('net', [
	{name: 'addPeer', api: 'net_addStaticPeer'},
	{name: 'peers', api: 'net_getPeers'},
]);
lemo.createAPI('tx', [
    {name: 'sendTx', api: 'tx_sendTx'},
]);
