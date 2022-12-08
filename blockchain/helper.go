package blockchain

import (
	"github.com/sunvim/dogesyncer/crypto"
	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/dogesyncer/rawdb"
	"github.com/sunvim/dogesyncer/types"
)

func ecrecoverFromHeader(h *types.Header) (types.Address, error) {
	// get the extra part that contains the seal
	extra, err := types.GetIbftExtra(h)
	if err != nil {
		return types.Address{}, err
	}
	// get the sig
	msg, err := types.CalculateHeaderHash(h)
	if err != nil {
		return types.Address{}, err
	}

	pub, err := crypto.RecoverPubkey(extra.Seal, crypto.Keccak256(msg))
	if err != nil {
		return types.Address{}, err
	}

	return crypto.PubKeyToAddress(pub), nil
}

func reset(db ethdb.Database) {
	latestHash, _ := rawdb.ReadHeadHash(db)
	header, _ := rawdb.ReadHeader(db, latestHash)
	lastHeader, _ := rawdb.ReadHeader(db, header.ParentHash)
	rawdb.WriteHeadHash(db, lastHeader.Hash)
	rawdb.WriteHeadNumber(db, lastHeader.Number)
}
