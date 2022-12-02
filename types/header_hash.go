package types

import (
	"github.com/dogechain-lab/fastrlp"
)

var (
	HeaderHash       func(h *Header) Hash
	marshalArenaPool fastrlp.ArenaPool
)

func init() {
	HeaderHash = defHeaderHash
}

func defHeaderHash(h *Header) (hash Hash) {
	msg, _ := CalculateHeaderHash(h)
	return BytesToHash(msg)
}

// ComputeHash computes the hash of the header
func (h *Header) ComputeHash() *Header {
	h.Hash = HeaderHash(h)
	return h
}
