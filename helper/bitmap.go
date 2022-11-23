package helper

import (
	"bytes"
	"sync"
)

var (
	BufPool = &sync.Pool{
		New: func() any {
			return &bytes.Buffer{}
		},
	}
)
