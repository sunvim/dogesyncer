package mdbx

import (
	"github.com/sunvim/dogesyncer/ethdb"
	"github.com/sunvim/gmdbx"
)

type MdbxDB struct {
	path string
	env  *gmdbx.Env
	dbi  map[string]gmdbx.DBI
}

var (
	defaultGeometry = gmdbx.Geometry{
		SizeLower:       1 << 21,
		SizeNow:         1 << 21,
		SizeUpper:       1 << 43,
		GrowthStep:      1 << 23,
		ShrinkThreshold: 1 << 24,
		PageSize:        1 << 16,
	}

	defaultFlags = gmdbx.EnvSyncDurable |
		gmdbx.EnvNoTLS |
		gmdbx.EnvWriteMap |
		gmdbx.EnvLIFOReclaim |
		gmdbx.EnvNoReadAhead |
		gmdbx.EnvCoalesce

	dbis = []string{
		ethdb.BodyDBI,
		ethdb.AssistDBI,
		ethdb.TrieDBI,
		ethdb.NumHashDBI,
		ethdb.TxesDBI,
		ethdb.HeadDBI,
		ethdb.TDDBI,
	}
)

func NewMDBX(path string) *MdbxDB {

	env, err := gmdbx.NewEnv()
	if err != gmdbx.ErrSuccess {
		panic(err)
	}

	if err := env.SetOption(gmdbx.OptMaxDB, 1024); err != gmdbx.ErrSuccess {
		panic(err)
	}

	if err := env.SetOption(gmdbx.OptRpAugmentLimit, 0x7fffFFFF); err != gmdbx.ErrSuccess {
		panic(err)
	}

	if err := env.SetOption(gmdbx.OptMaxReaders, 32000); err != gmdbx.ErrSuccess {
		panic(err)
	}

	if err := env.SetGeometry(defaultGeometry); err != gmdbx.ErrSuccess {
		panic(err)
	}

	if err = env.Open(path, defaultFlags, 0664); err != gmdbx.ErrSuccess {
		panic(err)
	}

	d := &MdbxDB{
		path: path,
		dbi:  make(map[string]gmdbx.DBI),
	}
	d.env = env

	tx := &gmdbx.Tx{}
	if err = env.Begin(tx, gmdbx.TxReadWrite); err != gmdbx.ErrSuccess {
		panic(err)
	}
	defer tx.Commit()

	// create or open all dbi
	for _, dbiName := range dbis {
		dbi, err := tx.OpenDBI(ethdb.TrieDBI, gmdbx.DBCreate)
		if err != gmdbx.ErrSuccess {
			panic(err)
		}
		d.dbi[dbiName] = dbi
	}

	return d
}
