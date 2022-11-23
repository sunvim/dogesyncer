package blockchain

import (
	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/dogesyncer/types"
)

type IBlockchain interface {
	Header() *types.Header
	SubscribeEvents() Subscription
	GetBlockByNumber(blockNumber uint64, full bool) (*types.Block, bool)
	ChainDB() ethdb.Database
}
