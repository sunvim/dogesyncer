package types

import (
	"fmt"
	"sync/atomic"

	"github.com/dogechain-lab/fastrlp"
	pq "github.com/sunvim/utils/priorityqueue"
)

type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
}

type Block struct {
	Header       *Header
	Transactions []*Transaction
	Uncles       []*Header

	// Cache
	size atomic.Value // *uint64
}

// priority queue interface
func (b *Block) Compare(other pq.Item) int {
	o := other.(*Block)
	if o.Header.Number < b.Header.Number {
		return 1
	} else if o.Header.Number == b.Header.Number {
		return 0
	}
	return -1
}

func (b *Block) Hash() Hash {
	return b.Header.Hash
}

func (b *Block) Number() uint64 {
	return b.Header.Number
}

func (b *Block) ParentHash() Hash {
	return b.Header.ParentHash
}

func (b *Block) Body() *Body {
	return &Body{
		Transactions: b.Transactions,
		Uncles:       b.Uncles,
	}
}

func (b *Block) Size() uint64 {
	sizePtr := b.size.Load()
	if sizePtr == nil {
		bytes := b.MarshalRLP()
		size := uint64(len(bytes))
		b.size.Store(&size)

		return size
	}

	sizeVal, ok := sizePtr.(*uint64)
	if !ok {
		return 0
	}

	return *sizeVal
}

func (b *Block) String() string {
	str := fmt.Sprintf(`Block(#%v):`, b.Number())

	return str
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		Header:       &cpy,
		Transactions: b.Transactions,
		Uncles:       b.Uncles,
	}
}

func (b *Block) MarshalRLP() []byte {
	return b.MarshalRLPTo(nil)
}

func (b *Block) MarshalRLPTo(dst []byte) []byte {
	return MarshalRLPTo(b.MarshalRLPWith, dst)
}

func (b *Block) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	vv := ar.NewArray()
	vv.Set(b.Header.MarshalRLPWith(ar))

	if len(b.Transactions) == 0 {
		vv.Set(ar.NewNullArray())
	} else {
		v0 := ar.NewArray()
		for _, tx := range b.Transactions {
			v0.Set(tx.MarshalRLPWith(ar))
		}
		vv.Set(v0)
	}

	if len(b.Uncles) == 0 {
		vv.Set(ar.NewNullArray())
	} else {
		v1 := ar.NewArray()
		for _, uncle := range b.Uncles {
			v1.Set(uncle.MarshalRLPWith(ar))
		}
		vv.Set(v1)
	}

	return vv
}

func (b *Block) UnmarshalRLP(input []byte) error {
	return UnmarshalRlp(b.UnmarshalRLPFrom, input)
}

func (b *Block) UnmarshalRLPFrom(p *fastrlp.Parser, v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}

	if len(elems) < 3 {
		return fmt.Errorf("incorrect number of elements to decode block, expected at least 3 but found %d",
			len(elems))
	}

	// header
	b.Header = &Header{}
	if err := b.Header.UnmarshalRLPFrom(p, elems[0]); err != nil {
		return err
	}

	// transactions
	txns, err := elems[1].GetElems()
	if err != nil {
		return err
	}

	for _, txn := range txns {
		bTxn := &Transaction{}
		if err := bTxn.UnmarshalRLPFrom(p, txn); err != nil {
			return err
		}

		b.Transactions = append(b.Transactions, bTxn)
	}

	// uncles
	uncles, err := elems[2].GetElems()
	if err != nil {
		return err
	}

	for _, uncle := range uncles {
		bUncle := &Header{}
		if err := bUncle.UnmarshalRLPFrom(p, uncle); err != nil {
			return err
		}

		b.Uncles = append(b.Uncles, bUncle)
	}

	return nil
}
