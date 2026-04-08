package ui

import (
	"time"
)

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
