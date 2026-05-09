package net

import (
	"encoding/json"
	"errors"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

type LobbyClient struct {
	conn       *websocket.Conn
	mu         sync.Mutex
	Username   string
	LocalIPs   []string
	OnClients  func([]LobbyUser)
	OnRouting  func(RoutingInfo)
	OnIncoming func(IncomingCall)
	OnError    func(error)
}

type LobbyUser struct {
	Username string `json:"username"`
	PublicIP string `json:"public_ip"`
}

type RoutingInfo struct {
	TargetUsername string   `json:"target_username"`
	RouteType      string   `json:"route_type"`
	TargetIPs      []string `json:"target_ips"`
	TargetPublic   string   `json:"target_public"`
	RelayAddress   string   `json:"relay_address"`
	SessionID      string   `json:"session_id"`
}

type IncomingCall struct {
	CallerUsername string   `json:"caller_username"`
	RouteType      string   `json:"route_type"`
	CallerIPs      []string `json:"caller_ips"`
	CallerPublic   string   `json:"caller_public"`
	RelayAddress   string   `json:"relay_address"`
	SessionID      string   `json:"session_id"`
}

type lobbyMsg struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type registerPayload struct {
	Username string   `json:"username"`
	LocalIPs []string `json:"local_ips"`
}

type callPayload struct {
	TargetUsername string `json:"target_username"`
}

func NewLobbyClient(serverURL, username string, localIPs []string) (*LobbyClient, error) {
	c, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return nil, err
	}
	lc := &LobbyClient{
		conn:     c,
		Username: username,
		LocalIPs: localIPs,
	}
	reg := map[string]interface{}{
		"type": "register",
		"payload": registerPayload{
			Username: username,
			LocalIPs: localIPs,
		},
	}
	if err := lc.writeJSON(reg); err != nil {
		c.Close()
		return nil, err
	}
	go lc.readLoop()
	return lc, nil
}

func (lc *LobbyClient) Call(targetUsername string) error {
	msg := map[string]interface{}{
		"type":    "call",
		"payload": callPayload{TargetUsername: targetUsername},
	}
	return lc.writeJSON(msg)
}

func (lc *LobbyClient) Close() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.conn != nil {
		lc.conn.Close()
		lc.conn = nil
	}
}

func (lc *LobbyClient) writeJSON(v interface{}) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.conn == nil {
		return errors.New("connection closed")
	}
	return lc.conn.WriteJSON(v)
}

func (lc *LobbyClient) readLoop() {
	for {
		_, data, err := lc.conn.ReadMessage()
		if err != nil {
			if lc.OnError != nil {
				lc.OnError(err)
			}
			return
		}
		var msg lobbyMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "clients":
			var users []LobbyUser
			json.Unmarshal(msg.Payload, &users)
			if lc.OnClients != nil {
				lc.OnClients(users)
			}
		case "call_routing":
			var ri RoutingInfo
			json.Unmarshal(msg.Payload, &ri)
			if lc.OnRouting != nil {
				lc.OnRouting(ri)
			}
		case "incoming_call":
			var ic IncomingCall
			json.Unmarshal(msg.Payload, &ic)
			if lc.OnIncoming != nil {
				lc.OnIncoming(ic)
			}
		}
	}
}

func GetLocalIPs() []string {
	addrs, _ := net.InterfaceAddrs()
	var ips []string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ips = append(ips, ipnet.IP.String())
		}
	}
	return ips
}
