package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jrhy/mast"
	"github.com/steveyen/gkvlite"
)

func syncIndex() {
	f, err := os.Create("data.db")
	s, err := gkvlite.NewStore(f)
	defer s.Flush()
	defer s.Close()
	defer f.Close()

	treeA := newTestTree(nil, &mast.RemoteConfig{StoreImmutablePartsWith: newDB(s.SetCollection("db1", nil))})
	treeB := newTestTree(nil, &mast.RemoteConfig{StoreImmutablePartsWith: newDB(s.SetCollection("db2", nil))})

	// add some things
	for i := 0; i < 100_000; i++ {
		treeA.Insert(context.Background(), fmt.Sprintf("a%d", i), "")
		treeB.Insert(context.Background(), fmt.Sprintf("b%d", i), "")
	}

	// find diff
	fmt.Println("Finding differences between trees...")
	now := time.Now()
	diff := 0
	cursor, err := treeA.StartDiff(context.Background(), treeB)
	if err != nil {
		panic(err)
	}

	for {
		item, err := cursor.NextEntry(context.Background())
		if err == mast.ErrNoMoreDiffs {
			break
		}
		if err != nil {
			panic(err)
		}
		if item.Type == mast.DiffType_Add {
			diff++
		}
	}

	fmt.Printf("Found %d differences\n", diff)
	fmt.Println("Sync duration:", time.Since(now))
}

type sqlitePersist struct {
	kv *gkvlite.Collection
}

// Load implements mast.Persist.
func (s *sqlitePersist) Load(ctx context.Context, key string) ([]byte, error) {
	return s.kv.Get([]byte(key))
}

// NodeURLPrefix implements mast.Persist.
func (s *sqlitePersist) NodeURLPrefix() string {
	return "fixed"
}

// Store implements mast.Persist.
func (s *sqlitePersist) Store(ctx context.Context, key string, value []byte) error {
	return s.kv.Set([]byte(key), value)
}

func newDB(c *gkvlite.Collection) mast.Persist {
	return &sqlitePersist{kv: c}
}

func newTestTree(
	rootOptions *mast.CreateRemoteOptions,
	cfg *mast.RemoteConfig,
) *mast.Mast {
	ctx := context.Background()
	root := mast.NewRoot(rootOptions)
	m, err := root.LoadMast(ctx, cfg)
	if err != nil {
		panic(err)
	}
	return m
}
