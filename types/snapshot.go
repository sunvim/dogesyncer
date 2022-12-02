package types

import (
	"github.com/vmihailenco/msgpack/v5"
)

// Vote defines the vote structure
type Vote struct {
	Validator Address
	Address   Address
	Authorize bool
}

// Equal checks if two votes are equal
func (v *Vote) Equal(vv *Vote) bool {
	if v.Validator != vv.Validator {
		return false
	}

	if v.Address != vv.Address {
		return false
	}

	if v.Authorize != vv.Authorize {
		return false
	}

	return true
}

// Copy makes a copy of the vote, and returns it
func (v *Vote) Copy() *Vote {
	vv := new(Vote)
	*vv = *v

	return vv
}

// Snapshot is the current state at a given point in time for validators and votes
type Snapshot struct {
	// block number when the snapshot was created
	Number uint64

	// block hash when the snapshot was created
	Hash string

	// votes casted in chronological order
	Votes []*Vote

	// current set of validators
	Set Validators
}

func (s *Snapshot) Marshal() ([]byte, error) {
	return msgpack.Marshal(s)
}

func (s *Snapshot) Unmarshal(input []byte) error {
	return msgpack.Unmarshal(input, s)
}
