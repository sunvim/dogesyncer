package ethdb

var (
	TrieDBI    = "trie"
	BodyDBI    = "block"
	HeadDBI    = "head"
	AssistDBI  = "assist"
	NumHashDBI = "numhash"
	TxesDBI    = "txes"
	TDDBI      = "td" // total difficulty
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

type Database interface {
	Setter
	Getter
	Closer
	Batch() Batch
}
