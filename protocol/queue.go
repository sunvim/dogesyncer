package protocol

import (
	"fmt"
	"sync"

	"github.com/sunvim/dogesyncer/helper"
	"github.com/sunvim/dogesyncer/types"
	"github.com/sunvim/gmdbx"
)

var (
	top    = []byte("top")
	length = []byte("len")
)

var (
	defaultGeometry = gmdbx.Geometry{
		SizeLower:       1 << 24,
		SizeNow:         1 << 24,
		SizeUpper:       1 << 43,
		GrowthStep:      1 << 25,
		ShrinkThreshold: 1 << 26,
		PageSize:        1 << 16,
	}

	defaultFlags = gmdbx.EnvSyncDurable | gmdbx.EnvNoReadAhead | gmdbx.EnvCoalesce
)

type Queue struct {
	lock   sync.Mutex
	env    *gmdbx.Env
	dbi    gmdbx.DBI
	top    uint64
	length uint64
}

func NewQueue(path string) (*Queue, error) {
	env, err := gmdbx.NewEnv()
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	if err := env.SetOption(gmdbx.OptMaxDB, 1); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	if err := env.SetOption(gmdbx.OptRpAugmentLimit, 0x7fffFFFF); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	if err := env.SetOption(gmdbx.OptMaxReaders, 24); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	if err = env.SetOption(gmdbx.OptMergeThreshold16Dot16Percent, 256); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	txnDpInitial, err := env.GetOption(gmdbx.OptTxnDpInitial)
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}
	if err = env.SetOption(gmdbx.OptTxnDpInitial, txnDpInitial*2); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	dpReserveLimit, err := env.GetOption(gmdbx.OptDpReserveLimit)
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}
	if err = env.SetOption(gmdbx.OptDpReserveLimit, dpReserveLimit*2); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	defaultDirtyPagesLimit, err := env.GetOption(gmdbx.OptTxnDpLimit)
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}
	if err = env.SetOption(gmdbx.OptTxnDpLimit, defaultDirtyPagesLimit*2); err != gmdbx.ErrSuccess { // default is RAM/42
		return nil, fmt.Errorf("%s", err.Error())
	}

	if err := env.SetGeometry(defaultGeometry); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	if err = env.Open(path, defaultFlags, 0664); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}
	q := &Queue{
		env: env,
	}

	tx := &gmdbx.Tx{}
	if err = env.Begin(tx, gmdbx.TxReadWrite); err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}
	defer tx.Commit()

	dbi, err := tx.OpenDBI("default", gmdbx.DBCreate)
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}
	q.dbi = dbi

	return q, err
}

func (q *Queue) Init() error {
	q.lock.Lock()
	defer q.lock.Unlock()

	tx := &gmdbx.Tx{}
	if err := q.env.Begin(tx, gmdbx.TxReadOnly); err != gmdbx.ErrSuccess {
		return fmt.Errorf("%s", err.Error())
	}
	defer tx.Commit()

	// init length value
	qlen := &gmdbx.Val{}
	key := gmdbx.Bytes(&length)
	err := tx.Get(q.dbi, &key, qlen)
	if err != gmdbx.ErrSuccess {
		q.length = 0
	} else {
		q.length = qlen.U64()
	}

	// init top value
	topKey := gmdbx.Bytes(&top)
	topVal := gmdbx.Val{}
	err = tx.Get(q.dbi, &topKey, &topVal)
	if err != gmdbx.ErrSuccess {
		q.top = 0
	} else {
		q.top = topVal.U64()
	}

	return nil
}

func (q *Queue) Put(num uint64, block *types.Block) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	tx := &gmdbx.Tx{}
	if err := q.env.Begin(tx, gmdbx.TxReadWrite); err != gmdbx.ErrSuccess {
		return fmt.Errorf("%s", err.Error())
	}
	defer tx.Commit()
	// set new length
	q.length += 1
	// set new top
	q.top = num
	// insert new block
	nums := helper.EncodeVarint(num)
	blockNum := gmdbx.Bytes(&nums)
	blocks := block.MarshalRLPTo(nil)
	blockVal := gmdbx.Bytes(&blocks)
	err := tx.Put(q.dbi, &blockNum, &blockVal, gmdbx.PutUpsert)
	if err != gmdbx.ErrSuccess {
		return fmt.Errorf("%s", err.Error())
	}

	return nil
}

func (q *Queue) Pop() (*types.Block, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	tx := &gmdbx.Tx{}
	err := q.env.Begin(tx, gmdbx.TxReadWrite)
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}
	defer tx.Commit()

	popNum := q.top - q.length + 1
	pops := helper.EncodeVarint(popNum)
	popKey := gmdbx.Bytes(&pops)
	popVal := gmdbx.Val{}
	// get block
	err = tx.Get(q.dbi, &popKey, &popVal)
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	popVal.Bytes()
	var b *types.Block
	berr := b.UnmarshalRLP(popVal.Bytes())
	if berr != nil {
		return nil, berr
	}

	// remove pop block
	err = tx.Delete(q.dbi, &popKey, &popVal)
	if err != gmdbx.ErrSuccess {
		return nil, fmt.Errorf("%s", err.Error())
	}

	return b, nil
}

var (
	ErrQueueNotFound = fmt.Errorf("not found")
)

func (q *Queue) Exist(num uint64) (bool, error) {
	tx := &gmdbx.Tx{}
	err := q.env.Begin(tx, gmdbx.TxReadOnly)
	if err != gmdbx.ErrSuccess {
		return false, fmt.Errorf("%s", err.Error())
	}

	defer tx.Commit()
	nums := helper.EncodeVarint(num)
	blockNum := gmdbx.Bytes(&nums)
	err = tx.Get(q.dbi, &blockNum, &gmdbx.Val{})
	if err != gmdbx.ErrSuccess {
		if err != gmdbx.ErrNotFound {
			return false, fmt.Errorf("%s", err.Error())
		}
		return false, ErrQueueNotFound
	}
	return true, nil
}

func (q *Queue) Len() uint64 {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.length
}

func (q *Queue) Min() uint64 {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.top - q.length + 1
}

func (q *Queue) Max() uint64 {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.top
}

func (q *Queue) Close() error {
	q.lock.Lock()
	defer q.lock.Unlock()

	tx := &gmdbx.Tx{}
	if err := q.env.Begin(tx, gmdbx.TxReadWrite); err != gmdbx.ErrSuccess {
		return fmt.Errorf("%s", err.Error())
	}
	defer tx.Commit()

	// init length value
	lens := helper.EncodeVarint(q.length)
	lenVal := gmdbx.Bytes(&lens)
	key := gmdbx.Bytes(&length)
	err := tx.Put(q.dbi, &key, &lenVal, gmdbx.PutUpsert)
	if err != gmdbx.ErrSuccess {
		return fmt.Errorf("%s", err.Error())
	}

	// init top value
	topKey := gmdbx.Bytes(&top)
	tops := helper.EncodeVarint(q.top)
	topVal := gmdbx.Bytes(&tops)
	err = tx.Put(q.dbi, &topKey, &topVal, gmdbx.PutUpsert)
	if err != gmdbx.ErrSuccess {
		return fmt.Errorf("%s", err.Error())
	}

	q.env.CloseDBI(q.dbi)
	q.env.Close(false)

	return nil
}
