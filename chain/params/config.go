package params

type ChainConfig struct {
	ChainID   int64 `json:"chainid"`
	Timeout   int64 `json:"timeout"`   // Number of timeout between blocks to produce millsecond
	SleepTime int64 `json:"sleeptime"` // Time of one block is produced and before ohter node begin produce another block millsecond

}
