// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"

	"github.com/agntcy/dir/mcp/server"
)

func main() {
	if err := server.Serve(context.Background()); err != nil {
		log.Fatal(err)
	}
}
