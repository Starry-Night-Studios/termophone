package net

import (
	"encoding/binary"
	"io"
	"log"

	"github.com/libp2p/go-libp2p/core/network"
)

// Writer takes the compressed Opus frames from the pipeline and sends them over the p2p stream
func Writer(stream network.Stream, sendCh <-chan []byte) {
	defer stream.Close()
	header := make([]byte, 4) // allocate once outside loop

	for chunk := range sendCh {
		binary.LittleEndian.PutUint32(header, uint32(len(chunk)))
		if _, err := stream.Write(header); err != nil {
			log.Println("P2P stream write error:", err)
			return
		}
		if _, err := stream.Write(chunk); err != nil {
			log.Println("P2P stream write error:", err)
			return
		}
	}
}

// Reader pulls binary from the p2p stream and pushes Opus frames to the receive pipeline
func Reader(stream network.Stream, recvCh chan<- []byte) {
	defer stream.Close()
	header := make([]byte, 4)

	// Pre-allocate a large buffer to avoid per-frame allocations
	bufPool := make([]byte, 1024*1024)

	for {
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
				log.Println("P2P stream read error:", err)
			}
			return
		}
		length := binary.LittleEndian.Uint32(header)
		if length == 0 || length > 1024*1024 {
			log.Printf("unexpected packet size %d, dropping connection", length)
			return
		}

		// Use a slice of the pre-allocated buffer
		payload := bufPool[:length]
		if _, err := io.ReadFull(stream, payload); err != nil {
			log.Println("P2P stream payload read error:", err)
			return
		}

		// Allocate exact size for the channel to avoid overwriting during processing
		frame := make([]byte, length)
		copy(frame, payload)

		select {
		case recvCh <- frame:
		default:
			log.Println("recv dropped frame")
		}
	}
}
