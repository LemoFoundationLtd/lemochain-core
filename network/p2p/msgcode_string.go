// Code generated by "stringer -type MsgCode"; DO NOT EDIT.

package p2p

import "strconv"

const _MsgCode_name = "HeartbeatMsgProHandshakeMsgLstStatusMsgGetLstStatusMsgBlockHashMsgTxsMsgGetBlocksMsgBlocksMsgConfirmMsgGetConfirmsMsgConfirmsMsgDiscoverReqMsgDiscoverResMsgGetBlocksWithChangeLogMsg"

var _MsgCode_index = [...]uint8{0, 12, 27, 39, 54, 66, 72, 84, 93, 103, 117, 128, 142, 156, 181}

func (i MsgCode) String() string {
	i -= 1
	if i >= MsgCode(len(_MsgCode_index)-1) {
		return "MsgCode(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _MsgCode_name[_MsgCode_index[i]:_MsgCode_index[i+1]]
}
