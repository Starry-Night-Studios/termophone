package net

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
)

const ProtocolID = "/termophone/audio/1.0.0"
const ControlProtocolID = "/termophone/control/1.0.0"

func getIdentity() (crypto.PrivKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(home, ".termophone")
	os.MkdirAll(baseDir, 0755)

	keyPath := filepath.Join(baseDir, "identity.key")

	if keyBytes, err := os.ReadFile(keyPath); err == nil {
		priv, err := crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal key: %v", err)
		}
		return priv, nil
	}

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	keyBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return priv, os.WriteFile(keyPath, keyBytes, 0600)
}

// SetupHost creates a new libp2p host, loads identity, attaches peerstore.
func SetupHost(ctx context.Context, listenPort int, username string) (host.Host, <-chan network.Stream, error) {
	priv, err := getIdentity()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize identity: %v", err)
	}

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", listenPort),
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
			fmt.Sprintf("/ip6/::/udp/%d/quic-v1", listenPort),
			fmt.Sprintf("/ip6/::/tcp/%d", listenPort),
		),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.UserAgent("termophone/"+username),
		libp2p.Identity(priv),
	)
	if err != nil {
		return nil, nil, err
	}

	streamCh := make(chan network.Stream, 1)

	h.SetStreamHandler(ProtocolID, func(s network.Stream) {
		log.Printf("Incoming audio connection from: %s", s.Conn().RemotePeer())
		select {
		case streamCh <- s:
		default:
			log.Println("Incoming audio stream dropped (already connected to a peer)")
			s.Reset()
		}
	})

	h.SetStreamHandler(VideoProtocolID, func(s network.Stream) {
		log.Printf("Incoming screen share connection from: %s", s.Conn().RemotePeer())
		if err := ReceiveScreenShare(ctx, s); err != nil {
			log.Printf("ReceiveScreenShare error: %v", err)
		}
	})

	log.Printf("libp2p Host Started! ID: %s", h.ID())
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID())
	}

	return h, streamCh, nil
}
