package mdbx

import (
	"errors"
	"sync"

	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/gmdbx"
)

var (
	txpool = &sync.Pool{
		New: func() any {
			return &gmdbx.Tx{}
		},
	}
)

func (d *MdbxDB) Set(dbi string, k []byte, v []byte) error {

	tx := txpool.Get().(*gmdbx.Tx)
	defer txpool.Put(tx)

	if err := d.env.Begin(tx, gmdbx.TxReadWrite); err != gmdbx.ErrSuccess {
		return errors.New("open tx failed")
	}
	defer tx.Commit()

	key, val := gmdbx.Bytes(&k), gmdbx.Bytes(&v)
	if err := tx.Put(d.dbi[dbi], &key, &val, gmdbx.PutUpsert); err != gmdbx.ErrSuccess {
		return errors.New("insert db failed")
	}

	return nil
}

func (d *MdbxDB) Get(dbi string, k []byte) ([]byte, bool, error) {

	tx := txpool.Get().(*gmdbx.Tx)
	defer txpool.Put(tx)

	if err := d.env.Begin(tx, gmdbx.TxReadOnly); err != gmdbx.ErrSuccess {
		return nil, false, errors.New("open tx failed")
	}
	defer tx.Commit()

	key := gmdbx.Bytes(&k)

	val := gmdbx.Val{}
	err := tx.Get(d.dbi[dbi], &key, &val)
	if err != gmdbx.ErrSuccess {
		if err != gmdbx.ErrNotFound {
			return nil, false, errors.New("get failed: " + err.Error())
		}
		return nil, false, nil
	}

	return val.Bytes(), true, nil

}

func (d *MdbxDB) Close() error {

	for _, v := range d.dbi {
		d.env.CloseDBI(v)
	}
	d.env.Close(false)

	return nil
}

func (d *MdbxDB) Batch() ethdb.Batch {
	return &KVBatch{env: d.env, dbi: d.dbi}
}
