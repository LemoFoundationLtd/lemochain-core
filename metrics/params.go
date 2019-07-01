package metrics

const LevelDBPrefix = "glemo/db/chaindata/"

var (
	// txpool
	txpoolModule             = "txpool"
	InvalidTx_counterName    = "txpool/DelInvalidTxs/invalid"
	TxpoolNumber_counterName = "txpool/totalTxNumber"
	// 告警条件
	Alarm_InvalidTx    int64 = 100   // 累计每隔100个错误交易则告警一次
	Alarm_TxpoolNumber int64 = 10000 // 交易池中的总交易数达到10000笔之后告警一次

	// tx
	txModule                 = "tx"
	VerifyFailedTx_meterName = "tx/VerifyTx/verifyFailed"
	// 告警条件
	Alarm_verifyFailedTx float64 = 0.5 // 验证交易失败的速率超过2秒1笔则开始报警

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
	// 告警条件
	Alarm_HandleBlocksMsg                 float64 = 50  // 调用handleBlocksMsg的速率大于50次/s
	Alarm_HandleGetBlocksMsg              float64 = 100 // 调用handleGetBlocksMsg的速率大于100次/s
	Alarm_HandleBlockHashMsg              float64 = 5   // 调用handleBlockHashMsg的速率大于5次/s
	Alarm_HandleGetConfirmsMsg            float64 = 50  // 调用handleGetConfirmsMsg的速率大于50次/s
	Alarm_HandleConfirmMsg                float64 = 10  // 调用handleConfirmMsg的速率大于10次/s
	Alarm_HandleGetBlocksWithChangeLogMsg float64 = 50  // 调用handleGetBlocksWithChangeLogMsg的速率大于50次/s
	Alarm_HandleDiscoverReqMsg            float64 = 5   // 调用handleDiscoverReqMsg的速率大于5次/s
	Alarm_HandleDiscoverResMsg            float64 = 5   // 调用handleDiscoverReqMsg的速率大于5次/s

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
	// 告警条件
	Alarm_BlockInsert float64 = 5 // Insert chain 所用平均时间大于5s
	Alarm_MineBlock   float64 = 8 // Mine Block 所用平均时间大于8s

	// p2p
	p2pModule                 = "p2p"
	PeerConnFailed_meterName  = "p2p/listenLoop/failedHandleConn"
	ReadMsgSuccess_timerName  = "p2p/readLoop/readMsgSuccess"  // 统计成功读取msg的timer
	ReadMsgFailed_timerName   = "p2p/readLoop/readMsgFailed"   // 统计读取msg失败的timer
	WriteMsgSuccess_timerName = "p2p/WriteMsg/writeMsgSuccess" // 统计写msg成功的timer
	WriteMsgFailed_timerName  = "p2p/WriteMsg/writeMsgFailed"  // 统计写msg失败的timer
	// 告警条件
	Alarm_PeerConnFailed  float64 = 5  // 远程peer连接失败的频率大于5次/s
	Alarm_ReadMsgSuccess  float64 = 20 // 读取接收到的message所用的平均时间大于20s
	Alarm_ReadMsgFailed   float64 = 5  // 读取接收到的message失败的频率大于5次/s
	Alarm_WriteMsgSuccess float64 = 15 // 写操作的平均用时超过15s
	Alarm_WriteMsgFailed  float64 = 5  // 写操作失败的频率超过了5次/s

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
