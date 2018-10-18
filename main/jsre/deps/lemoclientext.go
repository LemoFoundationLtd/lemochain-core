// package lemoclientext contains glemo specific lemo-client.js extensions.
package deps

var ExtModules = map[string]string{
	"net":     Net_JS,
	"chain":   Chain_JS,
	"mine":    Mine_JS,
	"account": Account_JS,
	"tx":      Tx_JS,
}

const Account_JS = `
lemojs._extend({
	property: 'account',
	methods: [
		new lemojs._extend.Method({
			name: 'newKeyPair',
			call: 'account_newKeyPair',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getBalance',
			call: 'account_getBalance',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'getAccount',
			call: 'account_getAccount',
			params: 1
		}),
	]
});
`

const Chain_JS = `
lemojs._extend({
	property: 'chain',
	methods: [
		new lemojs._extend.Method({
			name: 'getBlockByNumber',
			call: 'chain_getBlockByNumber',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'getBlockByHash',
			call: 'chain_getBlockByHash',
			params: 1
		}),
	],
	properties: [
		new lemojs._extend.Property({
			name: 'chainID',
			getter: 'chain_getChainID'
		}),
		new lemojs._extend.Property({
			name: 'genesis',
			getter: 'chain_getGenesis'
		}),
		new lemojs._extend.Property({
			name: 'getCurrentBlock',
			getter: 'chain_getCurrentBlock'
		}),
		new lemojs._extend.Property({
			name: 'getLatestStableBlock',
			getter: 'chain_getLatestStableBlock'
		}),
		new lemojs._extend.Property({
			name: 'currentHeight',
			getter: 'chain_getCurrentHeight'
		}),
		new lemojs._extend.Property({
			name: 'latestStableHeight',
			getter: 'chain_getLatestStableHeight'
		}),
	]
});
`

const Tx_JS = `
lemojs._extend({
	property: 'tx',
	methods: [
		new lemojs._extend.Method({
			name: 'sendTx',
			call: 'tx_sendTx',
			params: 1
		}),
	]
});
`

const Mine_JS = `
lemojs._extend({
	property: 'mine',
	methods: [
		new lemojs._extend.Method({
			name: 'start',
			call: 'mine_mineStart',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'stop',
			call: 'mine_mineStop',
			params: 0
		}),
	],
	properties: [
		new lemojs._extend.Property({
			name: 'mining',
			getter: 'mine_isMining'
		}),
		new lemojs._extend.Property({
			name: 'getLemoBase',
			getter: 'mine_getLemoBase'
		}),
	]
});
`

const Net_JS = `
lemojs._extend({
	property: 'net',
	methods: [
		new lemojs._extend.Method({
			name: 'addPeer',
			call: 'net_addStaticPeer',
			params: 1
		}),
	],
	properties: [
		new lemojs._extend.Property({
			name: 'peers',
			getter: 'net_getPeers'
		}),
		new lemojs._extend.Property({
			name: 'getNodeVersion',
			getter: 'net_getNodeVersion'
		}),
	]
});
`
