package net

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

const StateTopic = "termophone/state/1.0.0"

type PeerState struct {
	PeerID    string `json:"peer_id"`
	Username  string `json:"username"`
	IsMuted   bool   `json:"is_muted"`
	IsTyping  bool   `json:"is_typing"`
	UpdatedAt int64  `json:"updated_at"`
}

type StateService struct {
	ctx    context.Context
	cancel context.CancelFunc

	h     host.Host
	ps    *pubsub.PubSub
	topic *pubsub.Topic
	sub   *pubsub.Subscription

	username string
	muted    *atomic.Bool

	mu    sync.RWMutex
	peers map[peer.ID]PeerState
	local PeerState
}

func NewStateService(parent context.Context, h host.Host, username string, muted *atomic.Bool) (*StateService, error) {
	ctx, cancel := context.WithCancel(parent)

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		return nil, err
	}
	topic, err := ps.Join(StateTopic)
	if err != nil {
		cancel()
		return nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		topic.Close()
		cancel()
		return nil, err
	}

	s := &StateService{
		ctx:      ctx,
		cancel:   cancel,
		h:        h,
		ps:       ps,
		topic:    topic,
		sub:      sub,
		username: username,
		muted:    muted,
		peers:    make(map[peer.ID]PeerState),
	}

	s.local = PeerState{
		PeerID:   h.ID().String(),
		Username: username,
	}

	go s.readLoop()
	go s.publishLoop(3 * time.Second)
	go s.gcLoop(15 * time.Second)

	return s, nil
}

func (s *StateService) Close() {
	s.cancel()
	if s.sub != nil {
		s.sub.Cancel()
	}
	if s.topic != nil {
		s.topic.Close()
	}
}

func (s *StateService) publishLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			st := PeerState{
				PeerID:    s.h.ID().String(),
				Username:  s.username,
				IsMuted:   s.muted != nil && s.muted.Load(),
				IsTyping:  false,
				UpdatedAt: time.Now().UnixMilli(),
			}

			payload, err := json.Marshal(st)
			if err != nil {
				continue
			}
			if err := s.topic.Publish(s.ctx, payload); err != nil {
				log.Printf("state publish error: %v", err)
				continue
			}

			s.mu.Lock()
			s.local = st
			s.mu.Unlock()
		}
	}
}

func (s *StateService) readLoop() {
	for {
		msg, err := s.sub.Next(s.ctx)
		if err != nil {
			return
		}

		if msg.ReceivedFrom == s.h.ID() {
			continue
		}

		var st PeerState
		if err := json.Unmarshal(msg.Data, &st); err != nil {
			continue
		}

		pid, err := peer.Decode(st.PeerID)
		if err != nil {
			continue
		}

		if st.UpdatedAt == 0 {
			st.UpdatedAt = time.Now().UnixMilli()
		}

		s.mu.Lock()
		s.peers[pid] = st
		s.mu.Unlock()
	}
}

func (s *StateService) gcLoop(ttl time.Duration) {
	ticker := time.NewTicker(ttl)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-ttl).UnixMilli()
			s.mu.Lock()
			for pid, st := range s.peers {
				if st.UpdatedAt < cutoff {
					delete(s.peers, pid)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *StateService) Snapshot() map[peer.ID]PeerState {
	s.mu.RLock()
	out := make(map[peer.ID]PeerState, len(s.peers)+1)
	for k, v := range s.peers {
		out[k] = v
	}
	out[s.h.ID()] = s.local
	s.mu.RUnlock()
	return out
}

func (s *StateService) HostID() peer.ID {
	snap := s.Snapshot()
	ids := make([]peer.ID, 0, len(snap))
	for id := range snap {
		ids = append(ids, id)
	}
	return ElectHost(s.h.ID(), ids)
}

func ElectHost(localID peer.ID, connectedPeers []peer.ID) peer.ID {
	all := make([]string, 0, len(connectedPeers)+1)
	all = append(all, localID.String())
	for _, p := range connectedPeers {
		if p == localID {
			continue
		}
		all = append(all, p.String())
	}
	sort.Strings(all)
	hostID, err := peer.Decode(all[0])
	if err != nil {
		return localID
	}
	return hostID
}

func IsHost(localID peer.ID, connectedPeers []peer.ID) bool {
	return ElectHost(localID, connectedPeers) == localID
}
