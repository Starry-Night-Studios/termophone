package net

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/dchest/siphash"
)

const HeaderSize = 16

type RelayClient struct {
	conn      *net.UDPConn
	roomID    uint32
	secretKey uint64
	myID      uint8
	k0        uint64
	k1        uint64
}

// NewRelayClient connects to the relay and prepares the cryptographic keys.
// The roomID, secretKey, and relayIP will all be handed to the client by the Lobby.
func NewRelayClient(relayAddr string, roomID uint32, secretKey uint64, myID uint8) (*RelayClient, error) {
	addr, err := net.ResolveUDPAddr("udp", relayAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve relay address: %v", err)
	}

	// DialUDP acts like a "connected" UDP socket. 
	// The OS will optimize routing since the destination never changes.
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to relay: %v", err)
	}

	// Set a healthy buffer size to prevent dropping audio if the OS scheduler hiccups
	conn.SetReadBuffer(2 * 1024 * 1024)

	return &RelayClient{
		conn:      conn,
		roomID:    roomID,
		secretKey: secretKey,
		myID:      myID,
		k0:        secretKey,
		k1:        ^secretKey, // The same derivation the relay uses
	}, nil
}

// SendAudio takes your Opus payload, wraps it in the custom header, and fires it.
func (c *RelayClient) SendAudio(payload []byte, targetMask uint8) error {
	// 1. Calculate the SipHash signature of the payload
	signature := siphash.Hash(c.k0, c.k1, payload)

	// 2. Build the exact 16-byte header the Relay expects
	// [4 bytes RoomID][8 bytes SipHash][1 byte SenderID][1 byte TargetMask][2 bytes reserved]
	packet := make([]byte, HeaderSize+len(payload))
	
	binary.BigEndian.PutUint32(packet[0:4], c.roomID)
	binary.BigEndian.PutUint64(packet[4:12], signature)
	packet[12] = c.myID
	packet[13] = targetMask
	// bytes 14 and 15 are left as zero padding

	// 3. Append the raw audio data
	copy(packet[16:], payload)

	// 4. Fire and forget
	_, err := c.conn.Write(packet)
	return err
}

// StartListening spins up a dedicated goroutine that runs forever,
// stripping headers and feeding the pure audio payloads into your mixer channel.
func (c *RelayClient) StartListening(audioChan chan<- []byte) {
	go func() {
		// Zero-allocation receive buffer
		buffer := make([]byte, 1500)

		for {
			n, err := c.conn.Read(buffer)
			if err != nil || n < HeaderSize {
				continue // Drop malformed packets instantly
			}

			// We don't technically *need* to verify the SipHash here because 
			// the Relay already proved it wasn't garbage. But checking the RoomID 
			// protects us against stray UDP packets hitting our local port.
			incomingRoomID := binary.BigEndian.Uint32(buffer[0:4])
			if incomingRoomID != c.roomID {
				continue
			}

			// Slice out just the payload (zero-copy extraction)
			// We clone it before sending it to the channel so the next Read() 
			// doesn't overwrite the memory while the mixer is trying to play it.
			payload := make([]byte, n-HeaderSize)
			copy(payload, buffer[HeaderSize:n])

			// Ship it to the audio pipeline (mixer.go)
			select {
			case audioChan <- payload:
			default:
				// If the mixer channel is full, drop the frame. 
				// In real-time VoIP, late audio is worse than dropped audio.
			}
		}
	}()
}

// Close hangs up the connection
func (c *RelayClient) Close() {
	c.conn.Close()
}