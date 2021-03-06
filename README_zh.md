![Logo of the project](./logo.png)

# LemoChain
[![Build Status](https://travis-ci.org/LemoFoundationLtd/lemochain-core.svg?branch=master)](https://travis-ci.org/LemoFoundationLtd/lemochain-core)
[![code coverage](https://img.shields.io/coveralls/LemoFoundationLtd/lemochain-core.svg?style=flat-square)](https://coveralls.io/r/LemoFoundationLtd/lemochain-core)
[![gitter chat](https://img.shields.io/gitter/room/LemoFoundationLtd/lemochain-core.svg?style=flat-square)](https://gitter.im/LemoFoundationLtd/lemochain-core)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![GitHub license](https://img.shields.io/badge/license-LGPL3.0-blue.svg?style=flat-square)](https://github.com/LemoFoundationLtd/lemochain-core/blob/master/LICENSE)

LemoChain是一个通用的数据交易区块链，各种规模的公司可以将其结构化的商业数据货币化，在这个平台上进行交易。通过加强区块链与日常商业的相关性，LemoChain将加速区块链技术融入我们的日常生活。  
LemoChain独创的DPoVP共识机制具有高响应速度的特性，解决了区块链分布式网络响应速度慢，难以在各种应用场景落地的难题。  
lemochain-core项目是这种共识机制的Go语言实现，其控制台命令文档见[lemo-client](https://github.com/LemoFoundationLtd/lemo-client)。  

[中文版](https://github.com/LemoFoundationLtd/lemochain-core/blob/master/README_zh.md)  
[English](https://github.com/LemoFoundationLtd/lemochain-core/blob/master/README.md)


## 安装


### 配置编译环境
- 安装`golang`，1.13版及以上
- 下载`lemochain-core`源代码
- 因为`ECDSA`算法代码是由C语言编写，所以编译时会用到`GCC`，建议`windows`下安装`mingw`，其他系统请点击[GCC文档](https://gcc.gnu.org/install)

### 编译
```
cd lemochain-core\main
go build
```
> 注意: 编译目标程序需为64位

---

## 运行节点

### 配置文件
这是运行LemoChain必备的文件，位于datadir根目录下，名为：`config.json`
其中定义了初始出块节点列表和本节点的一些运行参数
```json
{
	"chainID": "1203",
	"deputyCount": "17",
	"sleepTime": "3000",
	"timeout": "10000",
	"termDuration": "1000000",
	"interimDuration":"1000",
	"connectionLimit":"20"
}
```
- `chainID` LemoChain的ID，不能为0
- `deputyCount` 参与共识的最大节点数量
- `sleepTime` 收到区块后等待一定时间后再出块，以免区块中没有交易（后续版本将会改为根据交易池状态决定是否出块）
- `timeout` 各节点出块的超时时间，不能小于3秒(3000毫秒)
- `termDuration` 两个快照块之间间隔的区块数
- `interimDuration` 过渡期区块数
- `connectionLimit` 最大连接数（代理节点、白名单除外）

### 节点白名单
节点启动后会自动连接这些节点，位于datadir根目录下，名为：`whitelist`  
其中每个节点占据一行，格式为`NodeID@IP:Port`。以下是LemoChain测试网络的节点：
```
7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43@149.28.25.8:7003
34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f@45.77.121.107:7003
c7021a9c903da38ed499f486dba4539fbe12b8878d43e566674beebd36746e77c827a2849db3c1289e0adf25fce294253be5e7c9bb65d0b94cf8a7ec34c91468@149.28.68.93:7007
```
### 节点黑名单
节点启动后会拒绝连接这些节点，位于datadir根目录下，名为：`blacklist`   
配置方式与以上的节点白名单相同。


### 命令行
通过`console`命令运行`glemo`可以启动一个内置的JavaScript控制台，通过这个控制台可以运行所有[SDK](https://github.com/LemoFoundationLtd/lemo-client)方法。包括与LemoChain网络进行交互；管理账号；发送交易；部署与执行智能合约，等等。
```
$ glemo console
```
这个命令将会启动一个节点连接到LemoChain网络，并开始同步区块数据。你也可以不带`console`参数启动，在之后通过`glemo attach`连接到这个节点上

指定数据存储目录
```
$ glemo console --datadir=path/to/custom/data/folder
```
