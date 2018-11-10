
# LemoChain

LemoChain是一个通用的数据交易区块链，各种规模的公司可以将其结构化的商业数据货币化，在这个平台上进行交易。通过加强区块链与日常商业的相关性，LemoChain将加速区块链技术融入我们的日常生活。  
LemoChain独创的DPoVP共识机制具有高响应速度的特性，解决了区块链分布式网络响应速度慢，难以在各种应用场景落地的难题。  
lemochain-go项目旨在展示这种共识机制的原理，验证其吞吐量和交易确认速度的提升。

[中文版](https://github.com/LemoFoundationLtd/lemochain-go/blob/master/README_zh.md)  
[English](https://github.com/LemoFoundationLtd/lemochain-go/blob/master/README.md)


## 安装步骤
配置编译环境
- 安装`golang`，1.10版及以上
- 在环境变量中配置工作目录`GOPATH`
- 安装`git`
- 在`GOPATH`工作目录下创建源码目录并拉取代码
    ```
    mkdir src\github.com\LemoFoundationLtd
    git clone https://github.com/LemoFoundationLtd/lemochain-go src\github.com\LemoFoundationLtd\lemochain-go
    ```
- 因为`ECDSA`算法代码是由C语言编写，所以编译时会用到`GCC`，建议`windows`下安装`mingw`，其他系统请点击[参考](http://gcc.gnu.org/)

- 编译glemo
    ```
    cd src\github.com\LemoFoundationLtd\lemochain-go\main
    go build
    ```
- Note: 编译目标程序需为64位


## 运行节点
通过`console`命令运行`glemo`可以启动一个内置的JavaScript控制台，通过这个控制台可以运行所有`lemo`提供的工具方法。包括与LemoChain网络进行交互；管理账号；发送交易；部署与执行智能合约，等等。
```
$ glemo console
```
这个命令将会启动一个节点连接到LemoChain网络，并开始同步区块数据。你也可以不带`console`参数启动，在之后通过`glemo attach`连接到这个节点上

指定数据存储目录
```
$ glemo console --datadir=path/to/custom/data/folder
```



## 定制LemoChain
### 配置私链创始块
LemoChain可以通过创始块配置文件(`genesis.json`)实现定制。
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
- `founder`  16亿预挖LEMO的持有账户
- `extraData` 创始块header中的一个字段，用来对链进行一些说明
- `gasLimit` 创始块交易费用上限，用来限制块大小
- `parentHash` 创始块的父块hash
- `timestamp` 创始块建立时间，精确到秒
- `deputyNodes` 初始的出块节点列表
- `minerAddress` 挖矿收益地址
- `nodeID` 节点NodeID
- `ip` 节点IP地址
- `port` 节点端口号
- `rank` 出块节点排名
- `votes` 出块节点得票数

填好上面这个JSON文件中的配置，我们需要在启动每个Lemo节点前对其进行初始化
```
glemo init path/to/genesis.json
```

### 配置文件
这是运行LemoChain必备的文件，位于datadir根目录下，名为：`config.json`
其中定义了初始出块节点列表和本节点的一些运行参数
```json
{
	"chainID": "1203",
	"sleepTime": "3000",
	"timeout": "10000"
}
```
- `chainID` 链id
- `sleepTime` 块最小间隔
- `timeout` 出块超时时间


### 启动共识节点
共识节点负责记账和生产区块。目前的配置流程还比较繁琐
1. 使用`glemo console`启动节点控制台
2. 创建矿工账号，输入密码，并记录账号地址
```
personal.newAccount()
```
3. 以共识节点模式启动并开始挖矿
```
glemo --mine <other flags>
```