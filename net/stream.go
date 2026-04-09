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
	header := make([]byte, 6) // length (4) + seq (2)
	var seq uint16

	for chunk := range sendCh {
		binary.LittleEndian.PutUint32(header[0:4], uint32(len(chunk)))
		binary.LittleEndian.PutUint16(header[4:6], seq)
		seq++
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
	header := make([]byte, 6)

	for {
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
				log.Println("P2P stream read error:", err)
			}
			return
		}
		length := binary.LittleEndian.Uint32(header[0:4])
		seq := binary.LittleEndian.Uint16(header[4:6])

		if length == 0 || length > 1024*1024 {
			log.Printf("unexpected packet size %d, dropping connection", length)
			return
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(stream, payload); err != nil {
			log.Println("P2P stream payload read error:", err)
			return
		}

		seqBytes := make([]byte, 2, length+2)
		binary.LittleEndian.PutUint16(seqBytes, seq)
		payloadWithSeq := append(seqBytes, payload...)

		select {
		case recvCh <- payloadWithSeq:
		default:
			log.Println("recv dropped frame")
		}
	}
}
