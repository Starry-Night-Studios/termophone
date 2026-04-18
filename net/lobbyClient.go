package net

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// The exact same JSON contracts as the server
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
type AuthChallenge struct {
	Nonce []byte `json:"nonce"`
}
type AuthResponse struct {
	PeerID    string `json:"peer_id"`
	Username  string `json:"username"`
	Signature []byte `json:"signature"`
}
type DialPayload struct {
	TargetID string `json:"target_id"`
}
type IncomingCall struct {
	CallerID   string `json:"caller_id"`
	CallerName string `json:"caller_name"`
}
type CallResponse struct {
	CallerID string `json:"caller_id"`
	Accepted bool   `json:"accepted"`
}
type RoomReady struct {
	RelayIP   string `json:"relay_ip"`
	RelayPort int    `json:"relay_port"`
	RoomID    uint32 `json:"room_id"`
	SecretKey uint64 `json:"secret_key"`
	MyID      uint8  `json:"my_id"`
	PeerName  string `json:"peer_name"`
}
type ErrorPayload struct {
	Reason string `json:"reason"`
}

type LobbyClient struct {
	Conn           *websocket.Conn
	IncomingCallCh chan IncomingCall
	RoomReadyCh    chan RoomReady
	ErrorCh        chan string

	privKey  crypto.PrivKey
	myID     peer.ID
	username string
}

func NewLobbyClient(lobbyURL, username string, privKey crypto.PrivKey, myID peer.ID) (*LobbyClient, error) {
	u, err := url.Parse(lobbyURL)
	if err != nil {
		return nil, err
	}
	// Auto-upgrade HTTP to WS
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}
	if !strings.HasSuffix(u.Path, "/ws") {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/ws"
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &LobbyClient{
		Conn:           conn,
		IncomingCallCh: make(chan IncomingCall, 5),
		RoomReadyCh:    make(chan RoomReady, 5),
		ErrorCh:        make(chan string, 5),
		privKey:        privKey,
		myID:           myID,
		username:       username,
	}

	go client.listen()
	return client, nil
}

func (c *LobbyClient) listen() {
	defer c.Conn.Close()
	defer close(c.RoomReadyCh)
	defer close(c.IncomingCallCh)
	defer close(c.ErrorCh)
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Lobby connection lost:", err)
			return
		}

		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			continue
		}

		switch env.Type {
		case "challenge":
			// THE GATEKEEPER: Solve the cryptographic puzzle
			var p AuthChallenge
			json.Unmarshal(env.Payload, &p)

			// Sign the server's random nonce with our Ed25519 Private Key
			sig, err := c.privKey.Sign(p.Nonce)
			if err != nil {
				log.Println("Failed to sign auth challenge:", err)
				continue
			}

			resp, _ := json.Marshal(Envelope{
				Type: "auth_response",
				Payload: toJSON(AuthResponse{
					PeerID:    c.myID.String(),
					Username:  c.username,
					Signature: sig,
				}),
			})
			c.Conn.WriteMessage(websocket.TextMessage, resp)

		case "incoming_call":
			var p IncomingCall
			json.Unmarshal(env.Payload, &p)
			c.IncomingCallCh <- p

		case "room_ready":
			var p RoomReady
			json.Unmarshal(env.Payload, &p)
			c.RoomReadyCh <- p

		case "call_failed":
			var p ErrorPayload
			json.Unmarshal(env.Payload, &p)
			c.ErrorCh <- p.Reason
		}
	}
}

// Dial tells the Lobby we want to call someone
func (c *LobbyClient) Dial(targetID string) {
	msg, _ := json.Marshal(Envelope{
		Type:    "dial",
		Payload: toJSON(DialPayload{TargetID: targetID}),
	})
	err := c.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("failed to send message to lobby:", err)
	}
}

// RespondToCall tells the Lobby if we hit Accept [Y] or Decline [N]
func (c *LobbyClient) RespondToCall(callerID string, accepted bool) {
	msg, _ := json.Marshal(Envelope{
		Type:    "call_response",
		Payload: toJSON(CallResponse{CallerID: callerID, Accepted: accepted}),
	})
	err := c.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("failed to send message to lobby:", err)
	}
}

func toJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
