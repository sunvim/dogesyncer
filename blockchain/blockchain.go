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
	"github.com/sunvim/dogesyncer/state"
	"github.com/sunvim/dogesyncer/types"
)

type Blockchain struct {
	logger            hclog.Logger
	config            *chain.Chain
	chaindb           ethdb.Database
	genesis           types.Hash
	stream            *eventStream // Event subscriptions
	executor          Executor
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

type Executor interface {
	BeginTxn(parentRoot types.Hash, header *types.Header, coinbase types.Address) (*state.Transition, error)
	//nolint:lll
	ProcessTransactions(transition *state.Transition, gasLimit uint64, transactions []*types.Transaction) (*state.Transition, error)
	Stop()
}

func NewBlockchain(logger hclog.Logger, db ethdb.Database, chain *chain.Chain, executor Executor) (*Blockchain, error) {
	b := &Blockchain{
		logger:   logger.Named("blockchain"),
		chaindb:  db,
		config:   chain,
		stream:   &eventStream{},
		executor: executor,
	}

	err := b.initCaches(32)
	if err != nil {
		return nil, err
	}

	b.stream.push(&Event{})
	return b, nil
}

func (b *Blockchain) CurrentTD() *big.Int {
	td, ok := b.currentDifficulty.Load().(*big.Int)
	if !ok {
		return nil
	}

	return td
}

func (b *Blockchain) GetTD(hash types.Hash) (*big.Int, bool) {
	return b.readTotalDifficulty(hash)
}

// get receitps by block header hash
func (b *Blockchain) GetReceiptsByHash(hash types.Hash) ([]*types.Receipt, error) {
	// read body
	bodies, err := rawdb.ReadBody(b.chaindb, hash)
	if err != nil {
		return nil, err
	}
	// read receipts
	receipts := make([]*types.Receipt, len(bodies))
	for i, tx := range bodies {
		receipt, err := rawdb.ReadReceipt(b.chaindb, tx)
		if err != nil {
			return nil, err
		}
		receipts[i] = receipt
	}

	return receipts, nil
}

func (b *Blockchain) GetBodyByHash(hash types.Hash) (*types.Body, bool) {
	// read body
	bodies, err := rawdb.ReadBody(b.chaindb, hash)
	if err != nil {
		return nil, false
	}
	// read transactions
	txes := make([]*types.Transaction, len(bodies))
	for i, txhash := range bodies {
		tx, err := rawdb.ReadTransaction(b.chaindb, txhash)
		if err != nil {
			return nil, false
		}
		txes[i] = tx
	}
	return &types.Body{
		Transactions: txes,
	}, false
}

func (b *Blockchain) GetHeaderByHash(hash types.Hash) (*types.Header, bool) {
	header, err := rawdb.ReadHeader(b.chaindb, hash)
	if err != nil {
		return nil, false
	}
	return header, true
}

func (b *Blockchain) GetHeaderByNumber(n uint64) (*types.Header, bool) {
	// read hash
	hash, ok := rawdb.ReadCanonicalHash(b.chaindb, n)
	if !ok {
		return nil, false
	}

	header, err := rawdb.ReadHeader(b.chaindb, hash)
	if err != nil {
		return nil, false
	}

	return header, true
}

func (b *Blockchain) WriteBlock(block *types.Block) error {
	fmt.Printf("block: %+v\n", block)
	return nil
}

func (b *Blockchain) VerifyFinalizedBlock(block *types.Block) error {
	return nil
}

func (b *Blockchain) CalculateGasLimit(number uint64) (uint64, error) {
	return 0, nil
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

	b.logger.Info("genesis", "hash", b.config.Genesis.Hash())

	return nil
}

func (b *Blockchain) WriteHeader(header *types.Header) error {
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

func (b *Blockchain) writeGenesis(genesis *chain.Genesis) error {

	header := genesis.GenesisHeader()
	header.ComputeHash()
	b.genesis = header.Hash

	if err := rawdb.WriteTD(b.chaindb, header.Hash, 1); err != nil {
		return fmt.Errorf("write td failed %v", err)
	}

	return b.WriteHeader(header)
}

func (b *Blockchain) VerifyHeader(header *types.Header) error {
	// check parent hash
	hash, ok := rawdb.ReadCanonicalHash(b.chaindb, header.Number-1)
	if !ok {
		return fmt.Errorf("not found block %d ", header.Number-1)
	}
	parent, err := rawdb.ReadHeader(b.chaindb, hash)
	if err != nil {
		return fmt.Errorf("get parent header %v", err)
	}
	if parent.Hash != header.ParentHash {
		return fmt.Errorf("unexpected header %s != %s %s", parent.Hash, header.ParentHash, parent.ComputeHash().Hash)
	}
	// check header self hash
	if header.Hash != types.HeaderHash(header) {
		return fmt.Errorf("header self check err %s != %s", header.Hash, types.HeaderHash(header))
	}
	return nil
}

func (b *Blockchain) addHeaderSnap(header *types.Header) error {

	extra, err := types.GetIbftExtra(header)
	if err != nil {
		return err
	}
	s := &types.Snapshot{
		Hash:   header.Hash.String(),
		Number: header.Number,
		Votes:  []*types.Vote{},
		Set:    extra.Validators,
	}

	return rawdb.WriteSnap(b.chaindb, header.Number, s)
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

	// Calculate the new total difficulty
	if err := rawdb.WriteTD(b.chaindb, newHeader.Hash, newHeader.Difficulty); err != nil {
		return nil, err
	}

	// Update the blockchain reference
	b.setCurHeader(newHeader, newHeader.Difficulty)

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
