package net

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
)

const ProtocolID = "/termophone/audio/1.0.0"
const ControlProtocolID = "/termophone/control/1.0.0"

func getIdentityAndStore(ctx context.Context) (crypto.PrivKey, datastore.Batching, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	baseDir := filepath.Join(home, ".termophone")
	os.MkdirAll(baseDir, 0755)

	// 1. Identity Key
	keyPath := filepath.Join(baseDir, "identity.key")
	var priv crypto.PrivKey

	if keyBytes, err := os.ReadFile(keyPath); err == nil {
		priv, err = crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal key: %v", err)
		}
	} else {
		priv, _, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		keyBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(keyPath, keyBytes, 0600); err != nil {
			return nil, nil, err
		}
	}

	// 2. Peerstore Datastore (Badger)
	storePath := filepath.Join(baseDir, "store")
	ds, err := badger.NewDatastore(storePath, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open badger datastore: %v", err)
	}

	return priv, ds, nil
}

// SetupHost creates a new libp2p host, loads identity, attaches peerstore, and runs Kad DHT.
func SetupHost(ctx context.Context, listenPort int, username string) (host.Host, *dht.IpfsDHT, datastore.Batching, <-chan network.Stream, error) {
        priv, ds, err := getIdentityAndStore(ctx)
        if err != nil {
                return nil, nil, nil, nil, fmt.Errorf("failed to initialize identity/store: %v", err)
        }

        ps, err := pstoreds.NewPeerstore(ctx, ds, pstoreds.DefaultOpts())
        if err != nil {
                return nil, nil, nil, nil, fmt.Errorf("failed to create persistent peerstore: %v", err)
        }

        h, err := libp2p.New(
                libp2p.ListenAddrStrings(
                        fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", listenPort),
                        fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
                ),
                libp2p.Transport(libp2pquic.NewTransport),
                libp2p.UserAgent("termophone/"+username),
                libp2p.Identity(priv),
                libp2p.Peerstore(ps),
        )
        if err != nil {
                return nil, nil, nil, nil, err
        }

        // Setup Kademlia DHT in client mode to avoid unnecessary WAN traffic
        kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeClient))
        if err != nil {
                return nil, nil, nil, nil, fmt.Errorf("failed to create DHT: %v", err)
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

        log.Printf("libp2p Host Started! ID: %s", h.ID())
        for _, addr := range h.Addrs() {
                log.Printf("  %s/p2p/%s", addr, h.ID())
        }

        return h, kadDHT, ds, streamCh, nil
}
