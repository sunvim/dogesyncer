package rawdb

import (
	"bufio"
	"bytes"
	"errors"

	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/dogesyncer/helper"
	"github.com/sunvim/dogesyncer/rlp"
	"github.com/sunvim/dogesyncer/types"
)

func ReadCanonicalHash(db ethdb.Database, number uint64) (types.Hash, bool) {
	v, ok, _ := db.Get(ethdb.NumHashDBI, helper.EncodeVarint(number))
	if !ok {
		return types.Hash{}, false
	}
	return types.BytesToHash(v), true
}

func WriteCanonicalHash(db ethdb.Database, number uint64, hash types.Hash) error {
	return db.Set(ethdb.NumHashDBI, helper.EncodeVarint(number), hash.Bytes())
}

func WriteHeader(db ethdb.Database, header *types.Header) error {
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		return err
	}
	return db.Set(ethdb.HeadDBI, header.Hash.Bytes(), data)
}

func ReadHeader(db ethdb.Database, hash types.Hash) (*types.Header, error) {
	header := &types.Header{}
	v, ok, err := db.Get(ethdb.HeadDBI, hash.Bytes())
	if err != nil {
		return header, err
	}

	if ok {
		err = rlp.DecodeBytes(v, header)
		if err != nil {
			return header, err
		}
	}

	return header, nil
}

func WriteBody(db ethdb.Database, hash types.Hash, body *types.Body) error {
	if len(body.Transactions) == 0 {
		return nil
	}
	buf := helper.BufPool.Get().(*bytes.Buffer)
	defer helper.BufPool.Put(buf)
	buf.Reset()

	for _, v := range body.Transactions {
		buf.Write(v.Hash().Bytes())
	}

	return db.Set(ethdb.BodyDBI, hash[:], buf.Bytes())

}

func ReadBody(db ethdb.Database, hash types.Hash) ([]types.Hash, error) {

	v, ok, err := db.Get(ethdb.BodyDBI, hash[:])
	if err != nil {
		return nil, err
	}

	if ok {
		buf := helper.BufPool.Get().(*bytes.Buffer)
		defer helper.BufPool.Put(buf)
		buf.Reset()
		buf.Write(v)
		scanner := bufio.NewScanner(buf)
		scanner.Split(ScanHash)
		var rs []types.Hash
		for scanner.Scan() {
			rs = append(rs, types.BytesToHash(scanner.Bytes()))
		}
		return rs, nil
	}
	return nil, errors.New("body not found")
}

func WriteTransaction(db ethdb.Database, tx *types.Transaction) error {
	return db.Set(ethdb.TxesDBI, tx.Hash().Bytes(), tx.MarshalRLPTo(nil))
}

func ReadTransaction(db ethdb.Database, hash types.Hash) (*types.Transaction, error) {
	v, ok, err := db.Get(ethdb.TxesDBI, hash.Bytes())
	if err != nil {
		return nil, err
	}

	if ok {
		tx := &types.Transaction{}
		err = tx.UnmarshalRLP(v)
		if err != nil {
			return nil, err
		}
		return tx, nil
	}
	return nil, errors.New("not found tx")
}
