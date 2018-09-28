// package lemoclientext contains glemo specific lemo-client.js extensions.
package deps

var ExtModules = map[string]string{
	"net":     Net_JS,
	"chain":   Chain_JS,
	"mine":    Mine_JS,
	"account": Account_JS,
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
			name: 'getVersion',
			call: 'account_getVersion',
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
			name: 'getBlock',
			call: 'chain_getBlock',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'getChainID',
			call: 'chain_getChainID',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getGenesis',
			call: 'chain_getGenesis',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getCurrentBlock',
			call: 'chain_getCurrentBlock',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'getStableBlock',
			call: 'chain_getStableBlock',
			params: 0
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
			call: 'mine_start',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'stop',
			call: 'mine_stop',
			params: 0
		}),
	],
	properties: [
		new lemojs._extend.Property({
			name: 'mining',
			getter: 'mine_isMining'
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
			getter: 'net_peers'
		}),
	]
});
`
