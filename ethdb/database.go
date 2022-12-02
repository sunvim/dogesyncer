package ethdb

var (
	TrieDBI     = "trie"
	BodyDBI     = "block"
	HeadDBI     = "head"
	AssistDBI   = "assist"
	NumHashDBI  = "numhash"
	TxesDBI     = "txes"
	ReceiptsDBI = "receipts"
	TDDBI       = "td"    // total difficulty
	SnapDBI     = "snap"  // consensus snapshot
	QueueDBI    = "queue" // cache sync block
)

type Setter interface {
	Set(dbi string, k, v []byte) error
}

type Getter interface {
	Get(dbi string, k []byte) ([]byte, bool, error)
}

type Batch interface {
	Setter
	Write() error
}

type Closer interface {
	Close() error
}

type Remover interface {
	Remove(dbi string, k []byte) error
}

type Database interface {
	Setter
	Getter
	Closer
	Remover
	Batch() Batch
}
