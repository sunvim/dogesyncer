package rawdb

import (
	"math/big"

	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/dogesyncer/helper"
	"github.com/sunvim/dogesyncer/types"
)

func ReadTD(db ethdb.Database, hash types.Hash) (*big.Int, bool) {
	v, ok, err := db.Get(ethdb.TDDBI, hash[:])
	if err != nil || !ok {
		return nil, false
	}

	x, _ := helper.DecodeVarint(v)
	return big.NewInt(int64(x)), true

}

func WriteTD(db ethdb.Database, hash types.Hash, number uint64) error {
	return db.Set(ethdb.TDDBI, hash[:], helper.EncodeVarint(number))
}

func ReadHeadNumber(db ethdb.Database) (uint64, bool) {

	v, ok, err := db.Get(ethdb.AssistDBI, latestBlockNumber)
	if err != nil {
		return 0, false
	}
	number, _ := helper.DecodeVarint(v)
	return number, ok

}

func WriteHeadNumber(db ethdb.Database, number uint64) error {
	return db.Set(ethdb.AssistDBI, latestBlockNumber, helper.EncodeVarint(number))
}

func ReadHeadHash(db ethdb.Database) (types.Hash, bool) {

	v, ok, err := db.Get(ethdb.AssistDBI, latestBlockHash)
	if err != nil {
		return types.Hash{}, false
	}
	return types.BytesToHash(v), ok

}

func WriteHeadHash(db ethdb.Database, hash types.Hash) error {
	return db.Set(ethdb.AssistDBI, latestBlockHash, hash.Bytes())
}

func WriteBlockByHash(db ethdb.Database, hash types.Hash, block *types.Block) error {

	return nil
}

func ReadBlockByHash(db ethdb.Database, hash types.Hash) (*types.Block, bool) {
	return nil, false
}
