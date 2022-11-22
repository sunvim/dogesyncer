package ethdb

var (
	TrieDBI  = "trie"
	BlockDBI = "block"
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
