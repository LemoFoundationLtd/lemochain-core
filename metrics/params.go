package metrics

const LevelDBPrefix = "glemo/db/chaindata/"

var (
	// txpool
	txpoolModule           = "txpool"
	RecvTx_meterName       = "txpool/RecvTx/receiveTx"
	InvalidTx_counterName  = "txpool/DelInvalidTxs/invalid"
	TxpoolNumber_gaugeName = "txpool/totalTxNumber"

	// tx
	txModule                 = "tx"
	VerifyFailedTx_meterName = "tx/VerifyTx/verifyFailed"

	// network
	networkModule                             = "network"
	HandleBlocksMsg_meterName                 = "network/protocol_manager/handleBlocksMsg"                 // 统计调用handleBlocksMsg的频率
	HandleGetBlocksMsg_meterName              = "network/protocol_manager/handleGetBlocksMsg"              // 统计调用handleGetBlocksMsg的频率
	HandleBlockHashMsg_meterName              = "network/protocol_manager/handleBlockHashMsg"              // 统计调用handleBlockHashMsg的频率
	HandleGetConfirmsMsg_meterName            = "network/protocol_manager/handleGetConfirmsMsg"            // 统计调用handleGetConfirmsMsg的频率
	HandleConfirmMsg_meterName                = "network/protocol_manager/handleConfirmMsg"                // 统计调用handleConfirmMsg的频率
	HandleGetBlocksWithChangeLogMsg_meterName = "network/protocol_manager/handleGetBlocksWithChangeLogMsg" // 统计调用handleGetBlocksWithChangeLogMsg的频率
	HandleDiscoverReqMsg_meterName            = "network/protocol_manager/handleDiscoverReqMsg"            // 统计调用handleDiscoverReqMsg的频率
	HandleDiscoverResMsg_meterName            = "network/protocol_manager/handleDiscoverResMsg"            // 统计调用handleDiscoverResMsg的频率

	// leveldb
	leveldbModule               = LevelDBPrefix
	LevelDb_get_timerName       = LevelDBPrefix + "user/gets"
	LevelDb_put_timerName       = LevelDBPrefix + "user/puts"
	LevelDb_del_timerName       = LevelDBPrefix + "user/dels"
	LevelDb_miss_meterName      = LevelDBPrefix + "user/misses" // 对数据库进行get操作失败的频率
	LevelDb_read_meterName      = LevelDBPrefix + "user/reads"  // get数据库出来的数据字节大小
	LevelDb_write_meterName     = LevelDBPrefix + "user/writes" // put进数据库的数据字节大小
	LevelDb_compTime_meteName   = LevelDBPrefix + "user/time"
	LevelDb_compRead_meterName  = LevelDBPrefix + "user/input"
	LevelDb_compWrite_meterName = LevelDBPrefix + "user/output"

	// consensus
	consensusModule       = "consensus"
	BlockInsert_timerName = "consensus/InsertBlock/insertBlock" // 统计区块插入链中的速率和所用时间的分布情况
	MineBlock_timerName   = "consensus/MineBlock/mineBlock"     // 统计出块速率和时间分布

	// p2p
	p2pModule                 = "p2p"
	PeerConnFailed_meterName  = "p2p/listenLoop/failedHandleConn"
	ReadMsgSuccess_timerName  = "p2p/readLoop/readMsgSuccess"  // 统计成功读取msg的timer
	ReadMsgFailed_timerName   = "p2p/readLoop/readMsgFailed"   // 统计读取msg失败的timer
	WriteMsgSuccess_timerName = "p2p/WriteMsg/writeMsgSuccess" // 统计写msg成功的timer
	WriteMsgFailed_timerName  = "p2p/WriteMsg/writeMsgFailed"  // 统计写msg失败的timer

	// system meter
	systemModule           = "system"
	System_memory_allocs   = "system/memory/allocs"   // 申请内存的次数
	System__memory_frees   = "system/memory/frees"    // 释放内存的次数
	System_memory_inuse    = "system/memory/inuse"    // 已申请且仍在使用的字节数
	System_memory_pauses   = "system/memory/pauses"   // GC总的暂停时间的循环缓冲
	System_disk_readCount  = "system/disk/readcount"  // 读磁盘操作次数
	System_disk_readData   = "system/disk/readdata"   // 读取的字节总数
	System_disk_writeCount = "system/disk/writecount" // 写磁盘操作次数
	System_disk_writeData  = "system/disk/writedata"  // 写的字节总数
)
