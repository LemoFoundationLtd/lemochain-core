// Code generated by "stringer -type MsgCode"; DO NOT EDIT.

package p2p

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[HeartbeatMsg-1]
	_ = x[ProHandshakeMsg-2]
	_ = x[LstStatusMsg-3]
	_ = x[GetLstStatusMsg-4]
	_ = x[BlockHashMsg-5]
	_ = x[TxsMsg-6]
	_ = x[GetBlocksMsg-7]
	_ = x[BlocksMsg-8]
	_ = x[ConfirmMsg-9]
	_ = x[GetConfirmsMsg-10]
	_ = x[ConfirmsMsg-11]
	_ = x[DiscoverReqMsg-12]
	_ = x[DiscoverResMsg-13]
	_ = x[GetBlocksWithChangeLogMsg-14]
}

const _MsgCode_name = "HeartbeatMsgProHandshakeMsgLstStatusMsgGetLstStatusMsgBlockHashMsgTxsMsgGetBlocksMsgBlocksMsgConfirmMsgGetConfirmsMsgConfirmsMsgDiscoverReqMsgDiscoverResMsgGetBlocksWithChangeLogMsg"

var _MsgCode_index = [...]uint8{0, 12, 27, 39, 54, 66, 72, 84, 93, 103, 117, 128, 142, 156, 181}

func (i MsgCode) String() string {
	i -= 1
	if i >= MsgCode(len(_MsgCode_index)-1) {
		return "MsgCode(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _MsgCode_name[_MsgCode_index[i]:_MsgCode_index[i+1]]
}
