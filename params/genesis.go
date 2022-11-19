package params

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/go-multierror"
	"github.com/sunvim/dogesyncer/types"
)

// Genesis specifies the header fields, state of a genesis block
type Genesis struct {
	Config *Params `json:"config"`

	Nonce      [8]byte                           `json:"nonce"`
	Timestamp  uint64                            `json:"timestamp"`
	ExtraData  []byte                            `json:"extraData,omitempty"`
	GasLimit   uint64                            `json:"gasLimit"`
	Difficulty uint64                            `json:"difficulty"`
	Mixhash    types.Hash                        `json:"mixHash"`
	Coinbase   types.Address                     `json:"coinbase"`
	Alloc      map[types.Address]*GenesisAccount `json:"alloc,omitempty"`

	// Override
	StateRoot types.Hash

	// Only for testing
	Number     uint64     `json:"number"`
	GasUsed    uint64     `json:"gasUsed"`
	ParentHash types.Hash `json:"parentHash"`
}

// Params are all the set of params for the chain
type Params struct {
	Forks          *Forks                 `json:"forks"`
	ChainID        int                    `json:"chainID"`
	Engine         map[string]interface{} `json:"engine"`
	BlockGasTarget uint64                 `json:"blockGasTarget"`
	BlackList      []string               `json:"blackList,omitempty"`
	DDOSPretection bool                   `json:"ddosPretection,omitempty"`
}

func (p *Params) GetEngine() string {
	// We know there is already one
	for k := range p.Engine {
		return k
	}

	return ""
}

// Forks specifies when each fork is activated
type Forks struct {
	Homestead      *Fork `json:"homestead,omitempty"`
	Byzantium      *Fork `json:"byzantium,omitempty"`
	Constantinople *Fork `json:"constantinople,omitempty"`
	Petersburg     *Fork `json:"petersburg,omitempty"`
	Istanbul       *Fork `json:"istanbul,omitempty"`
	EIP150         *Fork `json:"EIP150,omitempty"`
	EIP158         *Fork `json:"EIP158,omitempty"`
	EIP155         *Fork `json:"EIP155,omitempty"`
	Preportland    *Fork `json:"pre-portland,omitempty"` // test hardfork only in some test networks
	Portland       *Fork `json:"portland,omitempty"`     // bridge hardfork
	Detroit        *Fork `json:"detroit,omitempty"`      // pos hardfork
}

func (f *Forks) on(ff *Fork, block uint64) bool {
	if ff == nil {
		return false
	}

	return ff.On(block)
}

func (f *Forks) active(ff *Fork, block uint64) bool {
	if ff == nil {
		return false
	}

	return ff.Active(block)
}

func (f *Forks) IsHomestead(block uint64) bool {
	return f.active(f.Homestead, block)
}

func (f *Forks) IsByzantium(block uint64) bool {
	return f.active(f.Byzantium, block)
}

func (f *Forks) IsConstantinople(block uint64) bool {
	return f.active(f.Constantinople, block)
}

func (f *Forks) IsPetersburg(block uint64) bool {
	return f.active(f.Petersburg, block)
}

func (f *Forks) IsEIP150(block uint64) bool {
	return f.active(f.EIP150, block)
}

func (f *Forks) IsEIP158(block uint64) bool {
	return f.active(f.EIP158, block)
}

func (f *Forks) IsEIP155(block uint64) bool {
	return f.active(f.EIP155, block)
}

func (f *Forks) IsPortland(block uint64) bool {
	return f.active(f.Portland, block)
}

func (f *Forks) IsDetroit(block uint64) bool {
	return f.active(f.Detroit, block)
}

func (f *Forks) At(block uint64) ForksInTime {
	return ForksInTime{
		Homestead:      f.active(f.Homestead, block),
		Byzantium:      f.active(f.Byzantium, block),
		Constantinople: f.active(f.Constantinople, block),
		Petersburg:     f.active(f.Petersburg, block),
		Istanbul:       f.active(f.Istanbul, block),
		EIP150:         f.active(f.EIP150, block),
		EIP158:         f.active(f.EIP158, block),
		EIP155:         f.active(f.EIP155, block),
		Preportland:    f.active(f.Preportland, block),
		Portland:       f.active(f.Portland, block),
		Detroit:        f.active(f.Detroit, block),
	}
}

func (f *Forks) IsOnPreportland(block uint64) bool {
	return f.on(f.Preportland, block)
}

func (f *Forks) IsOnPortland(block uint64) bool {
	return f.on(f.Portland, block)
}

func (f *Forks) IsOnDetroit(block uint64) bool {
	return f.on(f.Detroit, block)
}

type Fork uint64

func NewFork(n uint64) *Fork {
	f := Fork(n)

	return &f
}

func (f Fork) On(block uint64) bool {
	return block == uint64(f)
}

func (f Fork) Active(block uint64) bool {
	return block >= uint64(f)
}

func (f Fork) Int() *big.Int {
	return big.NewInt(int64(f))
}

// Genesis alloc

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                    `json:"code,omitempty"`
	Storage    map[types.Hash]types.Hash `json:"storage,omitempty"`
	Balance    *big.Int                  `json:"balance,omitempty"`
	Nonce      uint64                    `json:"nonce,omitempty"`
	PrivateKey []byte                    `json:"secretKey,omitempty"` // for tests
}

type genesisAccountEncoder struct {
	Code       *string                   `json:"code,omitempty"`
	Storage    map[types.Hash]types.Hash `json:"storage,omitempty"`
	Balance    *string                   `json:"balance"`
	Nonce      *string                   `json:"nonce,omitempty"`
	PrivateKey *string                   `json:"secretKey,omitempty"`
}

// ENCODING //

func (g *GenesisAccount) MarshalJSON() ([]byte, error) {
	obj := &genesisAccountEncoder{}

	if g.Code != nil {
		obj.Code = types.EncodeBytes(g.Code)
	}

	if len(g.Storage) != 0 {
		obj.Storage = g.Storage
	}

	if g.Balance != nil {
		obj.Balance = types.EncodeBigInt(g.Balance)
	}

	if g.Nonce != 0 {
		obj.Nonce = types.EncodeUint64(g.Nonce)
	}

	if g.PrivateKey != nil {
		obj.PrivateKey = types.EncodeBytes(g.PrivateKey)
	}

	return json.Marshal(obj)
}

// DECODING //

func (g *GenesisAccount) UnmarshalJSON(data []byte) error {
	type GenesisAccount struct {
		Code       *string                   `json:"code,omitempty"`
		Storage    map[types.Hash]types.Hash `json:"storage,omitempty"`
		Balance    *string                   `json:"balance"`
		Nonce      *string                   `json:"nonce,omitempty"`
		PrivateKey *string                   `json:"secretKey,omitempty"`
	}

	var dec GenesisAccount
	if err := json.Unmarshal(data, &dec); err != nil {
		return err
	}

	var err error

	var subErr error

	parseError := func(field string, subErr error) {
		err = multierror.Append(err, fmt.Errorf("%s: %w", field, subErr))
	}

	if dec.Code != nil {
		g.Code, subErr = types.ParseBytes(dec.Code)
		if subErr != nil {
			parseError("code", subErr)
		}
	}

	if dec.Storage != nil {
		g.Storage = dec.Storage
	}

	g.Balance, subErr = types.ParseUint256orHex(dec.Balance)
	if subErr != nil {
		parseError("balance", subErr)
	}

	g.Nonce, subErr = types.ParseUint64orHex(dec.Nonce)

	if subErr != nil {
		parseError("nonce", subErr)
	}

	if dec.PrivateKey != nil {
		g.PrivateKey, subErr = types.ParseBytes(dec.PrivateKey)
		if subErr != nil {
			parseError("privatekey", subErr)
		}
	}

	return err
}
