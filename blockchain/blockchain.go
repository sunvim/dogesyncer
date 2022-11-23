package blockchain

import (
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/hashicorp/go-hclog"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sunvim/dogesyncer/chain"
	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/dogesyncer/rawdb"
	"github.com/sunvim/dogesyncer/types"
)

type Blockchain struct {
	logger            hclog.Logger
	config            *chain.Chain
	chaindb           ethdb.Database
	genesis           types.Hash
	stream            *eventStream // Event subscriptions
	currentHeader     atomic.Value // The current header
	currentDifficulty atomic.Value // The current difficulty of the chain (total difficulty)

	headersCache         *lru.Cache // LRU cache for the headers
	blockNumberHashCache *lru.Cache // LRU cache for the CanonicalHash
	difficultyCache      *lru.Cache // LRU cache for the difficulty
	// We need to keep track of block receipts between the verification phase
	// and the insertion phase of a new block coming in. To avoid having to
	// execute the transactions twice, we save the receipts from the initial execution
	// in a cache, so we can grab it later when inserting the block.
	// This is of course not an optimal solution - a better one would be to add
	// the receipts to the proposed block (like we do with Transactions and Uncles), but
	// that is currently not possible because it would break backwards compatibility due to
	// insane conditionals in the RLP unmarshal methods for the Block structure, which prevent
	// any new fields from being added
	receiptsCache *lru.Cache // LRU cache for the block receipts

}

func NewBlockchain(logger hclog.Logger, db ethdb.Database, chain *chain.Chain) (*Blockchain, error) {
	b := &Blockchain{
		logger:  logger.Named("blockchain"),
		chaindb: db,
		config:  chain,
		stream:  &eventStream{},
	}

	err := b.initCaches(32)
	if err != nil {
		return nil, err
	}

	b.stream.push(&Event{})
	return b, nil
}

// initCaches initializes the blockchain caches with the specified size
func (b *Blockchain) initCaches(size int) error {
	var err error

	b.headersCache, err = lru.New(size)
	if err != nil {
		return fmt.Errorf("unable to create headers cache, %w", err)
	}

	b.blockNumberHashCache, err = lru.New(size)
	if err != nil {
		return fmt.Errorf("unable to create canonical cache, %w", err)
	}

	b.difficultyCache, err = lru.New(size)
	if err != nil {
		return fmt.Errorf("unable to create difficulty cache, %w", err)
	}

	b.receiptsCache, err = lru.New(size)
	if err != nil {
		return fmt.Errorf("unable to create receipts cache, %w", err)
	}

	return nil
}

func (b *Blockchain) ChainDB() ethdb.Database {
	return b.chaindb
}

func (b *Blockchain) HandleGenesis() error {

	head, ok := rawdb.ReadHeadHash(b.chaindb)
	if ok { // non empty storage
		genesis, ok := rawdb.ReadCanonicalHash(b.chaindb, 0)
		if !ok {
			return fmt.Errorf("failed to load genesis hash")
		}
		// check genesis hash
		if genesis != b.config.Genesis.Hash() {
			return fmt.Errorf("genesis file does not match current genesis")
		}
		header, err := rawdb.ReadHeader(b.chaindb, head)
		if err != nil {
			return fmt.Errorf("failed to get header with hash %s err: %v", head.String(), err)
		}

		b.logger.Info("current header", "hash", head.String(), "number", header.Number)

		b.setCurHeader(header, header.Difficulty)

	} else { // empty storage, write the genesis

		if err := b.writeGenesis(b.config.Genesis); err != nil {
			return err
		}
	}

	b.logger.Info("genesis", "xhash", b.config.Genesis.Hash())

	return nil
}

func (b *Blockchain) writeGenesis(genesis *chain.Genesis) error {
	header := genesis.GenesisHeader()
	header.ComputeHash()

	b.genesis = header.Hash
	err := rawdb.WriteHeader(b.chaindb, header)
	if err != nil {
		return fmt.Errorf("failed to write header %s %v", header.Hash, err)
	}

	// Advance the head
	if _, err = b.advanceHead(header); err != nil {
		return err
	}

	// Create an event and send it to the stream
	event := &Event{}
	event.AddNewHeader(header)
	b.stream.push(event)

	return nil
}

func (b *Blockchain) advanceHead(newHeader *types.Header) (*big.Int, error) {
	err := rawdb.WriteHeadHash(b.chaindb, newHeader.Hash)
	if err != nil {
		return nil, err
	}

	err = rawdb.WriteHeadNumber(b.chaindb, newHeader.Number)
	if err != nil {
		return nil, err
	}

	err = rawdb.WriteCanonicalHash(b.chaindb, newHeader.Number, newHeader.Hash)
	if err != nil {
		return nil, err
	}

	// Check if there was a parent difficulty
	parentTD := big.NewInt(0)
	if newHeader.ParentHash != types.StringToHash("") {
		td, ok := b.readTotalDifficulty(newHeader.ParentHash)
		if !ok {
			return nil, fmt.Errorf("parent difficulty not found")
		}

		parentTD = td
	}
	// Calculate the new total difficulty
	newTD := big.NewInt(0).Add(parentTD, big.NewInt(0).SetUint64(newHeader.Difficulty))
	if err := rawdb.WriteTD(b.chaindb, newHeader.Hash, newTD.Uint64()); err != nil {
		return nil, err
	}

	// Update the blockchain reference
	b.setCurHeader(newHeader, newTD.Uint64())

	return nil, nil
}

func (b *Blockchain) readTotalDifficulty(headerHash types.Hash) (*big.Int, bool) {
	// Try to find the difficulty in the cache
	foundDifficulty, ok := b.difficultyCache.Get(headerHash)
	if ok {
		// Hit, return the difficulty
		fd, ok := foundDifficulty.(*big.Int)
		if !ok {
			return nil, false
		}

		return fd, true
	}

	// Miss, read the difficulty from the DB
	dbDifficulty, ok := rawdb.ReadTD(b.chaindb, headerHash)
	if !ok {
		return nil, false
	}

	// Update the difficulty cache
	b.difficultyCache.Add(headerHash, dbDifficulty)

	return dbDifficulty, true
}

func (b *Blockchain) setCurHeader(header *types.Header, diff uint64) {
	b.currentHeader.Store(header.Copy())
	b.currentDifficulty.Store(big.NewInt(int64(diff)))
}

func (b *Blockchain) Header() *types.Header {

	header, ok := b.currentHeader.Load().(*types.Header)
	if !ok {
		return nil
	}

	return header
}

func (b *Blockchain) GetBlockByNumber(blockNumber uint64, full bool) (*types.Block, bool) {
	blkHash, ok := rawdb.ReadCanonicalHash(b.chaindb, blockNumber)
	if !ok {
		return nil, false
	}
	return rawdb.ReadBlockByHash(b.chaindb, blkHash)
}

// SubscribeEvents returns a blockchain event subscription
func (b *Blockchain) SubscribeEvents() Subscription {
	return b.stream.subscribe()
}
