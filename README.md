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
- Install `MySQL` v5.3 (or any later version).

### Compiling
```
cd src\github.com\LemoFoundationLtd\lemochain-go\main
go build
```
> NOTE: Target platform should be x64

---

## Running

### configuration file
It is necessary for running LemoChain. It is located in `datadir` and named as `config.json`
It defines initial deputy node list and some running configuration about this node.
```json
{
	"chainID": "1203",
	"sleepTime": "3000",
	"timeout": "10000",
	"dbUri": "root:123456@tcp(127.0.0.1:3306)/lemochain?charset=utf8mb4",
	"dbDriver": "mysql"
}
```
- `chainID` The ID of LemoChain
- `sleepTime` Wait seconds to generation block for fear that there is no transactins in block
- `timeout` The maximum limit of block generation for every nodes
- `dbUri` The connection string of database. It is like `[USER_NAME]:[PASSWORD]@tcp([IP]:[PORT])/[DB_NAME]?charset=utf8mb4`
- `dbDriver` The type of database driver

chainID | description
---|---
1 | LemoChain main net
100 | LemoChain develop net

### whitelist file
The node will connect all nodes in this file automatically. It is located in `datadir` and named as `whitelist`  
Write each node address in a row. The format is `NodeID@IP:Port`. There are some LemoChain dev-net nodes below:
```
7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43@149.28.25.8:7003
34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f@45.77.121.107:7003
c7021a9c903da38ed499f486dba4539fbe12b8878d43e566674beebd36746e77c827a2849db3c1289e0adf25fce294253be5e7c9bb65d0b94cf8a7ec34c91468@149.28.68.93:7007
```

### command line
Start up LemoChain's built-in interactive JavaScript console, (via the trailing `console` subcommand) through which you can invoke all official [SDK](https://github.com/LemoFoundationLtd/lemo-client) methods. You can simply interact with the LemoChain network; create accounts; transfer funds; deploy and interact with contracts. To do so:
```
$ glemo console
```
This command will start a node to sync block datas from LemoChain network. The `console` subcommand is optional and if you leave it out you can always attach to an already running node with `glemo attach`.

Run with specific data directory
```
$ glemo console --datadir=path/to/custom/data/folder
```
