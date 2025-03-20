// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package mockstream

import (
	"bufio"
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func HandleStream(ctx context.Context, listenCh chan<- string) func(s network.Stream) {
	return func(s network.Stream) {
		// Create a buffer stream for non-blocking read and write.
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		go readData(ctx, rw, listenCh, s.Close)
		go writeData(ctx, rw)

		go func() {
			<-ctx.Done()
			s.Close()
		}()
	}
}

func StartDataStream(ctx context.Context, h host.Host, protoc string, listenCh chan<- string) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			for _, p := range h.Peerstore().Peers() {
				s, err := h.NewStream(ctx, p, protocol.ID(protoc))
				if err != nil {
					continue
				}

				rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
				go readData(ctx, rw, listenCh, s.Close)
				go writeData(ctx, rw)
			}
		}
	}
}

func readData(ctx context.Context, rw *bufio.ReadWriter, listenCh chan<- string, closeFn func() error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			str, _ := rw.ReadString('\n')
			listenCh <- str

			_ = closeFn()

			return
		}
	}
}

func writeData(ctx context.Context, rw *bufio.ReadWriter) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, _ = rw.WriteString("hello world\n")
			_ = rw.Flush()
		}
	}
}
