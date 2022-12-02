package types

import (
	"github.com/dogechain-lab/fastrlp"
)

type RLPMarshaler interface {
	MarshalRLPTo(dst []byte) []byte
}

type marshalRLPFunc func(ar *fastrlp.Arena) *fastrlp.Value

func MarshalRLPTo(obj marshalRLPFunc, dst []byte) []byte {
	ar := fastrlp.DefaultArenaPool.Get()
	dst = obj(ar).MarshalTo(dst)
	fastrlp.DefaultArenaPool.Put(ar)

	return dst
}
