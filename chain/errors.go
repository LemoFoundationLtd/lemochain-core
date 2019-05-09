package chain

import "errors"

var (
	ErrNoGenesis                = errors.New("can't get genesis block")
	ErrBlockNotExist            = errors.New("block not exist in local")
	ErrExistBlock               = errors.New("block exist in local")
	ErrParentNotExist           = errors.New("parent block not exist in local")
	ErrSaveBlock                = errors.New("save block to db error")
	ErrSaveAccount              = errors.New("save account error")
	ErrVerifyHeaderFailed       = errors.New("verify block's header error")
	ErrVerifyBlockFailed        = errors.New("verify block error")
	ErrSnapshoterIsNil          = errors.New("snapshoter is not initialised")
	ErrInvalidConfirmSigner     = errors.New("invalid confirm signer")
	ErrInvalidSignedConfirmInfo = errors.New("invalid signed data of confirm info")
	ErrSetStableBlockToDB       = errors.New("set stable block to db error")
	ErrMineGenesis              = errors.New("can not mine genesis block")
	ErrNotDeputy                = errors.New("not a deputy address in specific height")
)
