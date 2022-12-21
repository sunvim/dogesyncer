package mdbx

import (
	"bytes"
	"time"

	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/dogesyncer/helper"
	"github.com/torquem-ch/mdbx-go/mdbx"
)

func (d *MdbxDB) Set(dbi string, k []byte, v []byte) error {
	buf := strbuf.Get().(*bytes.Buffer)
	buf.Reset()
	buf.WriteString(dbi)
	buf.Write(k)

	d.mcache.Put(buf.Bytes(), v)
	strbuf.Put(buf)
	if d.mcache.Size() > memsize/4 && d.fcache == nil {
		d.worker.Submit(func() {
			// flush froze cache data to database
			d.flush()
		})
	}

	return nil

}

func (d *MdbxDB) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.fcache = d.mcache
	d.mcache = cachePool.Get().(*MemDB)

	var (
		info  *mdbx.EnvInfo
		count uint64
		err   error
	)
	stx := time.Now()
	d.env.Update(func(txn *mdbx.Txn) error {
		iter := d.fcache.NewIterator()
		for iter.Next() {
			err = txn.Put(d.dbi[helper.B2S(iter.key[:4])], iter.key[4:], iter.value, 0)
			if err != nil {
				panic(err)
			}
			count++
		}
		iter.Release()
		info, _ = d.env.Info(txn)
		return nil
	})

	d.logger.Info("flush", "keys", count, "readers", info.NumReaders, "mapsize", info.MapSize, "elapse", time.Since(stx))
	d.fcache.Reset()
	cachePool.Put(d.fcache)
	d.fcache = nil
}

func (d *MdbxDB) Get(dbi string, k []byte) ([]byte, bool, error) {
	buf := strbuf.Get().(*bytes.Buffer)
	defer strbuf.Put(buf)

	buf.Reset()
	buf.WriteString(dbi)
	buf.Write(k)

	// read main cache
	val, err := d.mcache.Get(buf.Bytes())
	if err == nil {
		return val, true, nil
	}

	// read frozen cache
	if d.fcache != nil {
		val, err = d.fcache.Get(buf.Bytes())
		if err == nil {
			return val, true, nil
		}
	}

	var (
		v []byte
		r bool
		e error
	)

	e = d.env.View(func(txn *mdbx.Txn) error {

		v, e = txn.Get(d.dbi[dbi], k)
		if e != nil {
			return e
		}
		return nil
	})

	if e != nil {
		if e == mdbx.NotFound {
			e = nil
			r = false
		}
	} else {
		r = true
	}

	return v, r, e
}

func (d *MdbxDB) Sync() error {
	d.env.Sync(true, false)
	return nil
}

func (d *MdbxDB) Close() error {
	d.worker.Stop()

	var (
		info  *mdbx.EnvInfo
		count uint64
		err   error
	)
	stx := time.Now()
	d.env.Update(func(txn *mdbx.Txn) error {
		//flush main cache
		iter := d.mcache.NewIterator()
		for iter.Next() {
			err = txn.Put(d.dbi[helper.B2S(iter.key[:4])], iter.key[4:], iter.value, 0)
			if err != nil {
				panic(err)
			}
			count++
		}
		iter.Release()
		// flush frozen cache
		if d.fcache != nil {
			iter = d.fcache.NewIterator()
			for iter.Next() {
				err = txn.Put(d.dbi[helper.B2S(iter.key[:4])], iter.key[4:], iter.value, 0)
				if err != nil {
					panic(err)
				}
				count++
			}
			iter.Release()
		}

		info, _ = d.env.Info(txn)
		return nil
	})

	d.logger.Info("flush", "keys", count, "readers", info.NumReaders, "mapsize", info.MapSize, "elapse", time.Since(stx))
	d.env.Sync(true, false)
	d.logger.Info("sync data to file", "elapse", time.Since(stx))

	for _, dbi := range d.dbi {
		d.env.CloseDBI(dbi)
	}
	d.env.Close()
	return nil
}

func (d *MdbxDB) Batch() ethdb.Batch {
	return &KVBatch{db: d}
}

func (d *MdbxDB) Remove(dbi string, k []byte) error {
	return d.env.Update(func(txn *mdbx.Txn) error {
		return txn.Del(d.dbi[dbi], k, nil)
	})
}
