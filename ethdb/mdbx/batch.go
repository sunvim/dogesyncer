package mdbx

import (
	"errors"

	"github.com/sunvim/gmdbx"
)

type keyvalue struct {
	dbi   string
	key   []byte
	value []byte
}

// KVBatch is a batch write for leveldb
type KVBatch struct {
	env    *gmdbx.Env
	dbi    map[string]gmdbx.DBI
	writes []keyvalue
	size   int
}

func copyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

func (b *KVBatch) Set(dbi string, k, v []byte) error {
	b.writes = append(b.writes, keyvalue{dbi, copyBytes(k), copyBytes(v)})
	b.size += len(k) + len(v)
	return nil
}

// why no error handle
func (b *KVBatch) Write() error {

	tx := txpool.Get().(*gmdbx.Tx)
	defer txpool.Put(tx)

	if err := b.env.Begin(tx, gmdbx.TxReadWrite); err != gmdbx.ErrSuccess {
		return errors.New("open tx failed " + err.Error())
	}
	defer tx.Commit()

	var (
		err      gmdbx.Error
		key, val gmdbx.Val
	)

	for _, keyvalue := range b.writes {

		key = gmdbx.Bytes(&keyvalue.key)

		if keyvalue.value == nil {
			err = tx.Put(b.dbi[keyvalue.dbi], &key, &gmdbx.Val{}, gmdbx.PutUpsert)
		} else {
			val = gmdbx.Bytes(&keyvalue.value)
			err = tx.Put(b.dbi[keyvalue.dbi], &key, &val, gmdbx.PutUpsert)
		}

		if err != gmdbx.ErrSuccess {
			return errors.New("insert failed: " + err.Error())
		}
	}

	return nil
}
