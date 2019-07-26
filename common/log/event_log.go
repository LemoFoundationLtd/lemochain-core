package log

import (
	"fmt"
)

// 用于事件推送
var eventTag = "[event log]"

const (
	TxEvent        = "[tx event]"
	ChainMineEvent = "[chain event]"
	NetworkEvent   = "[network event]"
	ConsensusEvent = "[consensus event]"
)

func Eventf(eventType, formatMsg string, values ...interface{}) {
	msg := fmt.Sprintf(formatMsg, values...)
	srvLog.Warn(eventTag + eventType + msg)
}

func Event(eventType, msg string, ctx ...interface{}) {
	srvLog.Warn(eventTag+eventType+msg, ctx...)
}
