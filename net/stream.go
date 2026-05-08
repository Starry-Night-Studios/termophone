package net

import (
	"encoding/binary"
	"io"
	"log"
)

// Writer takes compressed Opus frames from the pipeline and sends them over
// any ReadWriteCloser (libp2p stream or relay WebSocket wrapper).
func Writer(rwc io.ReadWriteCloser, sendCh <-chan []byte) {
	defer rwc.Close()
	var seq uint16

	for chunk := range sendCh {
		frame := make([]byte, 6+len(chunk))
		binary.LittleEndian.PutUint32(frame[0:4], uint32(len(chunk)))
		binary.LittleEndian.PutUint16(frame[4:6], seq)
		seq++
		copy(frame[6:], chunk)

		if _, err := rwc.Write(frame); err != nil {
			log.Println("P2P stream write error:", err)
			return
		}
	}
}

// Reader pulls binary from the connection and pushes Opus frames to the receive pipeline.
func Reader(rwc io.ReadWriteCloser, recvCh chan<- []byte) {
	defer rwc.Close()
	defer close(recvCh)
	header := make([]byte, 6) // length (4) + seq (2)

	for {
		if _, err := io.ReadFull(rwc, header); err != nil {
			if err != io.EOF {
				log.Println("P2P stream read error:", err)
			}
			return
		}
		length := binary.LittleEndian.Uint32(header[0:4])

		// Seq is extracted but must be prepended to the payload for RecvPipeline sequence tracking
		_ = binary.LittleEndian.Uint16(header[4:6])

		if length == 0 || length > 1024*1024 {
			log.Printf("unexpected packet size %d, dropping connection", length)
			return
		}

		// Allocate the exact size needed just once per frame
		frame := make([]byte, length+2)

		// Put the sequence number at the start
		copy(frame[0:2], header[4:6])

		// Read the payload directly from the connection into the rest of the frame
		if _, err := io.ReadFull(rwc, frame[2:]); err != nil {
			log.Println("P2P stream payload read error:", err)
			return
		}

		select {
		case recvCh <- frame:
		default:
			log.Println("recv dropped frame")
		}
	}
}