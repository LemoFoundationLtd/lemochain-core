package log

import (
	"fmt"
	"strings"
)

// 用于事件推送
var eventTag = "[event log]"

const (
	TxEvent        = "[tx]"
	ConsensusEvent = "[consensus]"
	NetworkEvent   = "[network]"
)

func Eventf(eventType, formatMsg string, values ...interface{}) {
	detail := fmt.Sprintf(formatMsg, values...)
	msg := strings.Join([]string{eventTag, eventType, detail}, "\t")
	srvLog.Info(msg)
}

func Event(eventType, msg string, ctx ...interface{}) {
	m := strings.Join([]string{eventTag, eventType, msg}, "\t")
	srvLog.Info(m, ctx...)
}
