package types

import (
	ipld "github.com/ipfs/go-ipld-format"
)

type DagAPI interface {
	ipld.DAGService
}
