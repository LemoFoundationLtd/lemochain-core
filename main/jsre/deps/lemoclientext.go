// Copyright 2016 The lemochain-go Authors
// This file is part of the lemochain-go library.
//
// The lemochain-go library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lemochain-go library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lemochain-go library. If not, see <http://www.gnu.org/licenses/>.

// package lemoclientext contains glemo specific lemo-client.js extensions.
package deps

var ExtModules = map[string]string{
	"admin":    Admin_JS,
	"debug":    Debug_JS,
	"lemo":     Lemo_JS,
	"miner":    Miner_JS,
	"net":      Net_JS,
	"personal": Personal_JS,
	"rpc":      RPC_JS,
	"txpool":   TxPool_JS,
	"chain":    Chain_JS,
	"mine":     Mine_JS,
}

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

const Admin_JS = `
lemojs._extend({
	property: 'admin',
	methods: [
		new lemojs._extend.Method({
			name: 'addPeer',
			call: 'admin_addPeer',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'removePeer',
			call: 'admin_removePeer',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 1,
			inputFormatter: [null]
		}),
		new lemojs._extend.Method({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'sleepBlocks',
			call: 'admin_sleepBlocks',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'startRPC',
			call: 'admin_startRPC',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new lemojs._extend.Method({
			name: 'stopRPC',
			call: 'admin_stopRPC'
		}),
		new lemojs._extend.Method({
			name: 'startWS',
			call: 'admin_startWS',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new lemojs._extend.Method({
			name: 'stopWS',
			call: 'admin_stopWS'
		}),
	],
	properties: [
		new lemojs._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo'
		}),
		new lemojs._extend.Property({
			name: 'peers',
			getter: 'admin_peers'
		}),
		new lemojs._extend.Property({
			name: 'datadir',
			getter: 'admin_datadir'
		}),
	]
});
`

const Debug_JS = `
lemojs._extend({
	property: 'debug',
	methods: [
		new lemojs._extend.Method({
			name: 'printBlock',
			call: 'debug_printBlock',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'getBlockRlp',
			call: 'debug_getBlockRlp',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'setHead',
			call: 'debug_setHead',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'seedHash',
			call: 'debug_seedHash',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'dumpBlock',
			call: 'debug_dumpBlock',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'chaindbProperty',
			call: 'debug_chaindbProperty',
			params: 1,
			outputFormatter: console.log
		}),
		new lemojs._extend.Method({
			name: 'chaindbCompact',
			call: 'debug_chaindbCompact',
		}),
		new lemojs._extend.Method({
			name: 'metrics',
			call: 'debug_metrics',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'verbosity',
			call: 'debug_verbosity',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'vmodule',
			call: 'debug_vmodule',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'backtraceAt',
			call: 'debug_backtraceAt',
			params: 1,
		}),
		new lemojs._extend.Method({
			name: 'stacks',
			call: 'debug_stacks',
			params: 0,
			outputFormatter: console.log
		}),
		new lemojs._extend.Method({
			name: 'freeOSMemory',
			call: 'debug_freeOSMemory',
			params: 0,
		}),
		new lemojs._extend.Method({
			name: 'setGCPercent',
			call: 'debug_setGCPercent',
			params: 1,
		}),
		new lemojs._extend.Method({
			name: 'memStats',
			call: 'debug_memStats',
			params: 0,
		}),
		new lemojs._extend.Method({
			name: 'gcStats',
			call: 'debug_gcStats',
			params: 0,
		}),
		new lemojs._extend.Method({
			name: 'cpuProfile',
			call: 'debug_cpuProfile',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'startCPUProfile',
			call: 'debug_startCPUProfile',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'stopCPUProfile',
			call: 'debug_stopCPUProfile',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'goTrace',
			call: 'debug_goTrace',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'startGoTrace',
			call: 'debug_startGoTrace',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'stopGoTrace',
			call: 'debug_stopGoTrace',
			params: 0
		}),
		new lemojs._extend.Method({
			name: 'blockProfile',
			call: 'debug_blockProfile',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'setBlockProfileRate',
			call: 'debug_setBlockProfileRate',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'writeBlockProfile',
			call: 'debug_writeBlockProfile',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'mutexProfile',
			call: 'debug_mutexProfile',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'setMutexProfileRate',
			call: 'debug_setMutexProfileRate',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'writeMutexProfile',
			call: 'debug_writeMutexProfile',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'writeMemProfile',
			call: 'debug_writeMemProfile',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'traceBlock',
			call: 'debug_traceBlock',
			params: 2,
			inputFormatter: [null, null]
		}),
		new lemojs._extend.Method({
			name: 'traceBlockFromFile',
			call: 'debug_traceBlockFromFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new lemojs._extend.Method({
			name: 'traceBlockByNumber',
			call: 'debug_traceBlockByNumber',
			params: 2,
			inputFormatter: [null, null]
		}),
		new lemojs._extend.Method({
			name: 'traceBlockByHash',
			call: 'debug_traceBlockByHash',
			params: 2,
			inputFormatter: [null, null]
		}),
		new lemojs._extend.Method({
			name: 'traceTransaction',
			call: 'debug_traceTransaction',
			params: 2,
			inputFormatter: [null, null]
		}),
		new lemojs._extend.Method({
			name: 'preimage',
			call: 'debug_preimage',
			params: 1,
			inputFormatter: [null]
		}),
		new lemojs._extend.Method({
			name: 'getBadBlocks',
			call: 'debug_getBadBlocks',
			params: 0,
		}),
		new lemojs._extend.Method({
			name: 'storageRangeAt',
			call: 'debug_storageRangeAt',
			params: 5,
		}),
		new lemojs._extend.Method({
			name: 'getModifiedAccountsByNumber',
			call: 'debug_getModifiedAccountsByNumber',
			params: 2,
			inputFormatter: [null, null],
		}),
		new lemojs._extend.Method({
			name: 'getModifiedAccountsByHash',
			call: 'debug_getModifiedAccountsByHash',
			params: 2,
			inputFormatter:[null, null],
		}),
	],
	properties: []
});
`

const Lemo_JS = `
lemojs._extend({
	property: 'lemo',
	methods: [
		new lemojs._extend.Method({
			name: 'sign',
			call: 'lemo_sign',
			params: 2,
			inputFormatter: [lemojs._extend.formatters.inputAddressFormatter, null]
		}),
		new lemojs._extend.Method({
			name: 'resend',
			call: 'lemo_resend',
			params: 3,
			inputFormatter: [lemojs._extend.formatters.inputTransactionFormatter, lemojs._extend.utils.fromDecimal, lemojs._extend.utils.fromDecimal]
		}),
		new lemojs._extend.Method({
			name: 'signTransaction',
			call: 'lemo_signTransaction',
			params: 1,
			inputFormatter: [lemojs._extend.formatters.inputTransactionFormatter]
		}),
		new lemojs._extend.Method({
			name: 'submitTransaction',
			call: 'lemo_submitTransaction',
			params: 1,
			inputFormatter: [lemojs._extend.formatters.inputTransactionFormatter]
		}),
		new lemojs._extend.Method({
			name: 'getRawTransaction',
			call: 'lemo_getRawTransactionByHash',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'getRawTransactionFromBlock',
			call: function(args) {
				return (lemojs._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'lemo_getRawTransactionByBlockHashAndIndex' : 'lemo_getRawTransactionByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [lemojs._extend.formatters.inputBlockNumberFormatter, lemojs._extend.utils.toHex]
		}),
	],
	properties: [
		new lemojs._extend.Property({
			name: 'pendingTransactions',
			getter: 'lemo_pendingTransactions',
			outputFormatter: function(txs) {
				var formatted = [];
				for (var i = 0; i < txs.length; i++) {
					formatted.push(lemojs._extend.formatters.outputTransactionFormatter(txs[i]));
					formatted[i].blockHash = null;
				}
				return formatted;
			}
		}),
	]
});
`

const Miner_JS = `
lemojs._extend({
	property: 'miner',
	methods: [
		new lemojs._extend.Method({
			name: 'start',
			call: 'miner_start',
			params: 1,
			inputFormatter: [null]
		}),
		new lemojs._extend.Method({
			name: 'stop',
			call: 'miner_stop'
		}),
		new lemojs._extend.Method({
			name: 'setLemobase',
			call: 'miner_setLemobase',
			params: 1,
			inputFormatter: [lemojs._extend.formatters.inputAddressFormatter]
		}),
		new lemojs._extend.Method({
			name: 'setExtra',
			call: 'miner_setExtra',
			params: 1
		}),
		new lemojs._extend.Method({
			name: 'setGasPrice',
			call: 'miner_setGasPrice',
			params: 1,
			inputFormatter: [lemojs._extend.utils.fromDecimal]
		}),
		new lemojs._extend.Method({
			name: 'getHashrate',
			call: 'miner_getHashrate'
		}),
	],
	properties: []
});
`

const Net_JS = `
lemojs._extend({
	property: 'net',
	methods: [],
	properties: [
		new lemojs._extend.Property({
			name: 'version',
			getter: 'net_version'
		}),
	]
});
`

const Personal_JS = `
lemojs._extend({
	property: 'personal',
	methods: [
		new lemojs._extend.Method({
			name: 'importRawKey',
			call: 'personal_importRawKey',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'sign',
			call: 'personal_sign',
			params: 3,
			inputFormatter: [null, lemojs._extend.formatters.inputAddressFormatter, null]
		}),
		new lemojs._extend.Method({
			name: 'ecRecover',
			call: 'personal_ecRecover',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'openWallet',
			call: 'personal_openWallet',
			params: 2
		}),
		new lemojs._extend.Method({
			name: 'deriveAccount',
			call: 'personal_deriveAccount',
			params: 3
		}),
		new lemojs._extend.Method({
			name: 'signTransaction',
			call: 'personal_signTransaction',
			params: 2,
			inputFormatter: [lemojs._extend.formatters.inputTransactionFormatter, null]
		}),
	],
	properties: [
		new lemojs._extend.Property({
			name: 'listWallets',
			getter: 'personal_listWallets'
		}),
	]
})
`

const RPC_JS = `
lemojs._extend({
	property: 'rpc',
	methods: [],
	properties: [
		new lemojs._extend.Property({
			name: 'modules',
			getter: 'rpc_modules'
		}),
	]
});
`

const TxPool_JS = `
lemojs._extend({
	property: 'txpool',
	methods: [],
	properties:
	[
		new lemojs._extend.Property({
			name: 'content',
			getter: 'txpool_content'
		}),
		new lemojs._extend.Property({
			name: 'inspect',
			getter: 'txpool_inspect'
		}),
		new lemojs._extend.Property({
			name: 'status',
			getter: 'txpool_status',
			outputFormatter: function(status) {
				status.pending = lemojs._extend.utils.toDecimal(status.pending);
				status.queued = lemojs._extend.utils.toDecimal(status.queued);
				return status;
			}
		}),
	]
});
`
