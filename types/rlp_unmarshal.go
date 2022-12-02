package types

import (
	"github.com/dogechain-lab/fastrlp"
)

type RLPUnmarshaler interface {
	UnmarshalRLP(input []byte) error
}

type unmarshalRLPFunc func(p *fastrlp.Parser, v *fastrlp.Value) error

func UnmarshalRlp(obj unmarshalRLPFunc, input []byte) error {
	pr := fastrlp.DefaultParserPool.Get()

	v, err := pr.Parse(input)
	if err != nil {
		fastrlp.DefaultParserPool.Put(pr)

		return err
	}

	if err := obj(pr, v); err != nil {
		fastrlp.DefaultParserPool.Put(pr)

		return err
	}

	fastrlp.DefaultParserPool.Put(pr)

	return nil
}
