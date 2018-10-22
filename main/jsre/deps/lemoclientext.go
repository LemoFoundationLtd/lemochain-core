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
			name: 'getBlockByHeight',
			call: 'chain_getBlockByHeight',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'getBlockByHash',
			call: 'chain_getBlockByHash',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'chainID',
			call: 'chain_getChainID',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'genesis',
			call: 'chain_getGenesis',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getCurrentBlock',
			call: 'chain_getCurrentBlock',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getLatestStableBlock',
			call: 'chain_getLatestStableBlock',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'currentHeight',
			call: 'chain_getCurrentHeight',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'latestStableHeight',
			call: 'chain_getLatestStableHeight',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'gasPriceAdvice',
			call: 'chain_gasPriceAdvice',
			params: 0
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
		new lemojs._extend.Method({
			name: 'mining',
			call: 'mine_isMining',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getLemoBase',
			call: 'mine_getLemoBase',
			params: 0
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
		new lemojs._extend.Method({
			name: 'dropPeer',
			call: 'net_dropPeer',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'peers',
			call: 'net_getPeers',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getNodeVersion',
			call: 'net_getNodeVersion',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getListening',
			call: 'net_getListening',
			params: 0
		}),
	]
});
`
