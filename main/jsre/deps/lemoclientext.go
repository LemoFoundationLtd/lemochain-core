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
			name: 'newAccount',
			call: 'account_newAccount',
			params: 0
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
	],
	properties: [
		new lemojs._extend.Property({
			name: 'chainID',
			getter: 'chain_chainID'
		}),
		new lemojs._extend.Property({
			name: 'currentBlock',
			getter: 'chain_currentBlock'
		}),
		new lemojs._extend.Property({
			name: 'stableBlock',
			getter: 'chain_stableBlock'
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
