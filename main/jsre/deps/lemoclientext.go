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
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'getBlockByHash',
			call: 'chain_getBlockByHash',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'chainID',
			call: 'chain_chainID',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'genesis',
			call: 'chain_genesis',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'currentBlock',
			call: 'chain_currentBlock',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'latestStableBlock',
			call: 'chain_latestStableBlock',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'currentHeight',
			call: 'chain_currentHeight',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'latestStableHeight',
			call: 'chain_latestStableHeight',
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
			name: 'lemoBase',
			call: 'mine_lemoBase',
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
			call: 'net_peers',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'nodeVersion',
			call: 'net_nodeVersion',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'info',
			call: 'net_info',
			params: 0
		}),
	]
});
`
