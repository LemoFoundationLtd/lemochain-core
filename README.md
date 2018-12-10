![Logo of the project](./logo.png)

# LemoChain
[![Build Status](https://travis-ci.org/LemoFoundationLtd/lemochain-go.svg?branch=master)](https://travis-ci.org/LemoFoundationLtd/lemochain-go)
[![Coverage Status](https://coveralls.io/repos/github/LemoFoundationLtd/lemochain-go/badge.svg?branch=master)](https://coveralls.io/github/LemoFoundationLtd/lemochain-go?branch=master)
[![gitter chat](https://img.shields.io/gitter/room/LemoFoundationLtd/lemochain-go.svg?style=flat-square)](https://gitter.im/LemoFoundationLtd/lemochain-go)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![GitHub license](https://img.shields.io/badge/license-LGPL3.0-blue.svg?style=flat-square)](https://github.com/LemoFoundationLtd/lemochain-go/blob/master/LICENSE)

LemoChain is a data exchange blockchain, where companies of all sizes can monetize their structured business data and trade within the platform. By strengthening the relevance of the blockchain and daily business, LemoChain will accelerate the integration of blockchain technology into our daily lives.  
The original DPoVP consensus mechanism of LemoChain has the characteristic of high scalability, which solves the problem of the slow response of the existing distributed networks and the difficulties they face in complying with various application scenarios.  
The lemochain-go project aims to demonstrate the principle of this consensus mechanism, verifying its improved throughput and transaction confirmation speed.  
The lemochain-go project is the Golang implement of this consensus mechanism. [lemo-client](https://github.com/LemoFoundationLtd/lemo-client) is document of the command in lemochain-go console.  

[中文版](https://github.com/LemoFoundationLtd/lemochain-go/blob/master/README_zh.md)  
[English](https://github.com/LemoFoundationLtd/lemochain-go/blob/master/README.md)


## Installing

### Setup build tools
- Install `golang`, ensure the Go version is 1.10(or any later version).
- Set up the Path environment variable `GOPATH`
- Install `git`
- Make source code directory in `GOPATH` and download source code into it
    ```
    mkdir src\github.com\LemoFoundationLtd
    git clone https://github.com/LemoFoundationLtd/lemochain-go src\github.com\LemoFoundationLtd\lemochain-go
    ```
- Install `GCC`, cause ECDSA is required. Install `mingw` if you use windows, otherwise click [here](https://gcc.gnu.org/install) to read the GCC documentation.

### Compiling
```
cd src\github.com\LemoFoundationLtd\lemochain-go\main
go build
```
> NOTE: Target platform should be x64

---

## Running
Start up LemoChain's built-in interactive JavaScript console, (via the trailing `console` subcommand) through which you can invoke all official [SDK](https://github.com/LemoFoundationLtd/lemo-client) methods. You can simply interact with the LemoChain network; create accounts; transfer funds; deploy and interact with contracts. To do so:
```
$ glemo console
```
This command will start a node to sync block datas from LemoChain network. The `console` subcommand is optional and if you leave it out you can always attach to an already running node with `glemo attach`.

Run with specific data directory
```
$ glemo console --datadir=path/to/custom/data/folder
```

---

## Operating a private network

### Defining the private genesis state
First, you'll need to create the genesis state of your networks, which all nodes need to be aware of and agree upon. This consists of a small JSON file (e.g. call it `genesis.json`):
```json
{
  "founder": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		},
		{
			"minerAddress": "Lemo842BJZ4DKCC764C63Y6A943775JH6NQ3Z33Y",
			"nodeID": "0x7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43",
			"ip": "127.0.0.1",
			"port": 7003,
			"rank": 2,
			"votes": 15
		}
	]
}
```
- `founder`  The owner of the 1.6 billion pre-miner LEMO.
- `extraData` A property in genesis block's header, it is used to store some description about the chain.
- `gasLimit` The transactions' gas limit of genesis block, it is used to limit the number of transactions in one block.
- `parentHash` The parent block's hash of genesis block.
- `timestamp` The genesis creation timestamp seconds.
- `deputyNodes` Initial deputy node list
	- `minerAddress` The account address to receive mining benefit
	- `nodeID` The LemoChain node ID, it is from the public key whose private key is using for sign blocks
	- `ip` Deputy node IP address
	- `port` The port to connect other nodes
	- `rank` The rank of all deputy nodes
	- `votes` The votes count

With the genesis state defined in the above JSON file, you'll need to initialize every glemo node with it prior to starting it up to ensure all blockchain parameters are correctly set:
```
glemo init path/to/genesis.json
```

### configuration file
It is necessary for running LemoChain. It is located in `datadir` and named as `config.json`
It defines initial deputy node list and some running configuration about this node.
```json
{
	"chainID": "1203",
	"sleepTime": "3000",
	"timeout": "10000"
}
```
- `chainID` The ID of LemoChain
- `sleepTime` Wait seconds to generation block for fear that there is no transactins in block
- `timeout` The maximum limit of block generation for every nodes
chainID | description
---|---
1 | LemoChain main net
100 | LemoChain develop net

### Running nodes
Deputy nodes confirm transactions and produce blocks.
1. Run glemo with `console` command.
```
glemo console
```
2. Create miner benefit account to receive mining benefit
```
// Backup `private` to safe place
// Fill `address` with `founder` of `genesis.json`. So you can get the pre-mined LEMO
lemo.account.newKeyPair()
```
3. Create miner signing account. (You can use the account in step 2, but it's not safe)
```
// Save `private` as file `nodekey` in datadir
lemo.account.newKeyPair()
```
4. Set deputy node information in `deputyNodes` of `genesis.json`
```
{
	"minerAddress": "", // The address of miner benefit account
	"nodeID": "", // The public of miner signing account
	"ip": "127.0.0.1", // Deputy node IP address
	"port": 7001, // The port to connect other nodes
	"rank": 0, // The initial rank of all deputy nodes
	"votes": 17 // The initial votes count
}
```
5. Remove the directory `chaindata` in datadir, restart node
```
glemo console
```
Start mining
```
lemo.mine.start()
```
6. You can check or transfer LEMO in your account. More command is in JS SDK[Documentation](https://github.com/LemoFoundationLtd/lemo-client)
7. Make a whitelist file. The node will connect all nodes in this file automatically.  
Make a file named `whitelist` in datadir, write each node address in a row:
```
45.77.121.107:7003
149.28.25.8:7003
149.28.68.93:7003
```
Note：Now we need set node IP and port only, but the format will be changed to `NodeID@IP:Port` in next versions