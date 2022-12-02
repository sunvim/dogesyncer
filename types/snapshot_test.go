package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot(t *testing.T) {
	s := &Snapshot{
		Number: 100,
		Hash:   "0xedb014b780a9725ac77b7ba1dd2011d40562a3e825168f08ea1ca38ecade217b",
		Votes: []*Vote{
			{
				Validator: StringToAddress("0xe876adee133b67e413a2a4bc4d06c5d2a3ca7e63"),
				Address:   StringToAddress("0xe64b65f994bd085229e17b30aee5393d29210fea"),
				Authorize: true,
			},
		},
		Set: Validators{StringToAddress("0x1033c85d761ae13b996dc5a5d0f00acdc9529e19")},
	}
	b, err := s.Marshal()
	if err != nil {
		t.Error(err)
	}
	as := &Snapshot{}

	err = as.Unmarshal(b)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, s, as, "should be equal")
}
