package helper

import (
	"sync"

	"github.com/dogechain-lab/fastrlp"
)

var (
	RlpPool = &sync.Pool{
		New: func() any {
			return &fastrlp.Arena{}
		},
	}
)
