package main

import (
	"context"
	"fmt"
	"github.com/agntcy/dir/cli/cmd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer func() {
		cancel()
	}()

	if err := cmd.Run(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
