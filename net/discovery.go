package net

import (
	"context"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const DiscoveryServiceTag = "termophone-local-discovery"

type discoveryNotifee struct {
	h      host.Host
	peerCh chan<- peer.AddrInfo
}

// HandlePeerFound is triggered exactly when mDNS discovers another termophone node
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	// Ignore ourselves
	if pi.ID == n.h.ID() {
		return
	}

	// Always cache addresses so dialCb can use them locally.
	n.h.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.TempAddrTTL)

	// Ship to the UI selector!
	select {
	case n.peerCh <- pi:
	default:
		// Queue full or no listener
	}
}

// SetupDiscovery starts the mDNS beacon service
func SetupDiscovery(ctx context.Context, h host.Host, peerCh chan<- peer.AddrInfo) error {
	svc := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h, peerCh: peerCh})
	err := svc.Start()
	if err != nil {
		return err
	}
	log.Println("mDNS Auto-Discovery running...")
	return nil
}
