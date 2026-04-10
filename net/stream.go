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
	var seq uint16

	for chunk := range sendCh {
		frame := make([]byte, 6+len(chunk))
		binary.LittleEndian.PutUint32(frame[0:4], uint32(len(chunk)))
		binary.LittleEndian.PutUint16(frame[4:6], seq)
		seq++
		copy(frame[6:], chunk)

		if _, err := stream.Write(frame); err != nil {
			log.Println("P2P stream write error:", err)
			return
		}
	}
}

// Reader pulls binary from the p2p stream and pushes Opus frames to the receive pipeline
func Reader(stream network.Stream, recvCh chan<- []byte) {
	defer stream.Close()
	header := make([]byte, 6) // length (4) + seq (2)

	// Pre-allocate a large buffer to avoid per-frame allocations
	bufPool := make([]byte, 1024*1024)

	for {
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
				log.Println("P2P stream read error:", err)
			}
			return
		}
		length := binary.LittleEndian.Uint32(header[0:4])
		// seq is extracted but it must be prepended to the payload for RecvPipeline
		// sequence tracking which expects: [seq: 2 bytes][opus payload]
		_ = binary.LittleEndian.Uint16(header[4:6])

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

		// Allocate exact size for the channel including the 2-byte seq header
		// This avoids overwriting the bufPool during concurrent channel processing
		frame := make([]byte, length+2)
		copy(frame[0:2], header[4:6])
		copy(frame[2:], payload)

		select {
		case recvCh <- frame:
		default:
			log.Println("recv dropped frame")
		}
	}
}
