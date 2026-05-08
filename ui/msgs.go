package ui

import "time"

type MsgAudioLevel struct {
	Local float64
	Peer  float64
}

type MsgPeerConnected struct {
	Name string
	ID   string
}

type MsgPeerDisconnected struct{}

type MsgLog struct {
	Line string
}

type MsgStats struct {
	LossPercent float64
	LatencyMs   int
}

type MsgTick time.Time

// LobbyUser represents a user visible on the lobby server.
type LobbyUser struct {
	Username string
	PublicIP string
}

// MsgLobbyUsers is delivered to the UI whenever the lobby broadcasts its client list.
type MsgLobbyUsers struct {
	Users []LobbyUser
}