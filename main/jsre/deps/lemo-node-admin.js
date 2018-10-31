lemo._createAPI('account', [
	{name: 'newKeyPair', method: 'account_newKeyPair'},
]);
lemo._createAPI('mine', [
	{name: 'start', method: 'mine_mineStart'},
	{name: 'stop', method: 'mine_mineStop'},
]);
lemo._createAPI('net', [
	{name: 'addPeer', method: 'net_addStaticPeer'},
	{name: 'dropPeer', method: 'net_dropPeer'},
	{name: 'getPeers', method: 'net_peers'},
]);
