
# LemoChain

LemoChain is a data exchange blockchain, where companies of all sizes can monetize their structured business data and trade within the platform. By strengthening the relevance of the blockchain and daily business, LemoChain will accelerate the integration of blockchain technology into our daily lives.
The original DPoVP consensus mechanism of LemoChain has the characteristic of high scalability, which solves the problem of the slow response of the existing distributed networks and the difficulties they face in complying with various application scenarios.
The lemochain-go project aims to demonstrate the principle of this consensus mechanism, verifying its improved throughput and transaction confirmation speed.

[中文版](https://github.com/LemoFoundationLtd/lemochain-go/blob/master/README_zh.md)  
[English](https://github.com/LemoFoundationLtd/lemochain-go/blob/master/README.md)


## Installation Instructions
Setup build tools
- Install `golang`, ensure the Go version is 1.10(or any later version).
- Install `git`
- Install `mingw` if you use windows to build, cause GCC is required
- Set up the Path environment variable `GOPATH`
- Make source code directory in `GOPATH` and download source code into it
    ```
    mkdir src\github.com\LemoFoundationLtd
    git clone https://github.com/LemoFoundationLtd/lemochain-go src\github.com\LemoFoundationLtd\lemochain-go
    ```
- Compile glemo
    ```
    cd src\github.com\LemoFoundationLtd\lemochain-go
    go install -v ./glemo
    ```



## Running glemo
Start up LemoChain's built-in interactive JavaScript console, (via the trailing `console` subcommand) through which you can invoke all official `lemo` methods. You can simply interact with the LemoChain network; create accounts; transfer funds; deploy and interact with contracts. To do so:
```
$ glemo console
```
This command will start a node to sync block datas from LemoChain network. The `console` subcommand is optional and if you leave it out you can always attach to an already running node with `glemo attach`.

Run with specific data directory
```
$ glemo console --datadir=path/to/custom/data/folder
```



## Operating a private network
### Defining the private genesis state
First, you'll need to create the genesis state of your networks, which all nodes need to be aware of and agree upon. This consists of a small JSON file (e.g. call it `genesis.json`):
```json
{
  "lemobase": "0x0000000000000000000000000000000000000000",
  "extraData": "",
  "gasLimit": "0x2fefd8",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp": "0x00"
}
```
- `lemobase`  The owner of the 1.6 billion pre-miner LEMO.
- `extraData` A property in genesis block's header, it is used to store some description about the chain.
- `gasLimit` The transactions' gas limit of genesis block, it is used to limit the number of transactions in one block.
- `parentHash` The parent block's hash of genesis block.
- `timestamp` The genesis creation timestamp seconds.

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
	"timeout": "10000",
	"deputyNodes":[
		{
			"lemoBase": "0x4412441244124412441244124412441244124412",
			"nodeID": "0x04595e0e74b214824247d4ed71b826e887b6934aa1191e098d6188e0b097c433d5d9a21f2ffcbae790cc88c942db5c08df6cf4b3b67a2fb6ad0b952f19030446d4",
			"ip": "127.0.0.1",
			"port": "1234",
			"rank":"1",
			"votes":"123456"
		},
		{
			"lemoBase": "0x4412441244124412441244124412441244124413",
			"nodeID": "0x4689",
			"ip": "10.0.0.1",
			"port": "1234",
			"rank":"2",
			"votes":"3412"
		}
	]
}
```
- `chainID` The id of chain
- `sleepTime` The minimum limit of block generation for this node
- `timeout` The maximum limit of block generation for every nodes
- `deputyNodes` Initial deputy node list
- `lemoBase` The account who receive deputy benefit
- `nodeID` The id of this node
- `ip` The IP of this node
- `port` The port of this node
- `rank` Then rank of this deputy
- `votes` The votes of this deputy

### Running nodes
Star nodes confirm transactions and produce blocks.
1. Run glemo with `console` command.
2. Create account, enter the password and record address.
```
personal.newAccount()
```
3. Run glemo with deputy mode and start to mine
```
glemo --mine <other flags>
```
