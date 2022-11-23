package mdbx

import (
	"fmt"

	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/gmdbx"
)

func (d *MdbxDB) Set(dbi string, k []byte, v []byte) error {

	tx := &gmdbx.Tx{}
	if err := d.env.Begin(tx, gmdbx.TxReadWrite); err != gmdbx.ErrSuccess {
		return fmt.Errorf("open tx failed")
	}
	defer tx.Commit()

	key, val := gmdbx.Bytes(&k), gmdbx.Bytes(&v)
	if err := tx.Put(d.dbi[dbi], &key, &val, gmdbx.PutUpsert); err != gmdbx.ErrSuccess {
		return fmt.Errorf("insert db failed: " + err.Error())
	}

	return nil
}

func (d *MdbxDB) Get(dbi string, k []byte) ([]byte, bool, error) {

	tx := &gmdbx.Tx{}

	if err := d.env.Begin(tx, gmdbx.TxReadOnly); err != gmdbx.ErrSuccess {
		return nil, false, fmt.Errorf("open tx failed")
	}
	defer tx.Commit()

	key := gmdbx.Bytes(&k)

	val := gmdbx.Val{}
	err := tx.Get(d.dbi[dbi], &key, &val)
	if err != gmdbx.ErrSuccess {
		if err != gmdbx.ErrNotFound {
			return nil, false, fmt.Errorf("get failed: " + err.Error())
		}
		return nil, false, nil
	}

	return val.Bytes(), true, nil

}

func (d *MdbxDB) Close() error {

	for _, v := range d.dbi {
		d.env.CloseDBI(v)
	}
	if err := d.env.Close(false); err != gmdbx.ErrSuccess {
		return fmt.Errorf("close db failed: %s ", err.Error())
	}

	return nil
}

func (d *MdbxDB) Batch() ethdb.Batch {
	return &KVBatch{env: d.env, dbi: d.dbi}
}
