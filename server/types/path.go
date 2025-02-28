package types

import "github.com/ipfs/go-cid"

type ImmutablePath interface {
	String() string
	Namespace() string
	Mutable() bool
	Segments() []string
	RootCid() cid.Cid
}
