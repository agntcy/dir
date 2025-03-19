// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"bufio"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/network"
)

func handleStream(s network.Stream) {
	log.Println("Got a new stream!")

	// Create a buffer stream for non-blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	go writeData(rw)

	//s.Close()
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			fmt.Println(str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	for i := range 10 {
		rw.WriteString(fmt.Sprintf("%d\n", i))
		rw.Flush()
	}
	rw.WriteString("\n") // close
	rw.Flush()
}
