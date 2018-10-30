package chain

import "errors"

var (
	ErrNoGenesis                     = errors.New("can't get genesis block")
	ErrBlockNotExist                 = errors.New("block not exist in local")
	ErrSaveBlock                     = errors.New("save block to db error")
	ErrSaveAccount                   = errors.New("save account error")
	ErrVerifyBlockFailed             = errors.New("verify block error")
	ErrInvalidConfirmInfo            = errors.New("invalid confirm info")
	ErrInvalidSignedConfirmInfo      = errors.New("invalid signed data of confirm info")
	ErrSetConfirmInfoToDB            = errors.New("set confirm info to db error")
	ErrSetStableBlockToDB            = errors.New("set stable block to db error")
	ErrStableHeightLargerThanCurrent = errors.New("stable block's height is larger than current block")
)
