package net

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
)

func mustPeerID(t *testing.T, s string) peer.ID {
	t.Helper()
	id, err := peer.Decode(s)
	if err != nil {
		t.Fatalf("failed to decode peer id: %v", err)
	}
	return id
}

func TestElectHost_PicksLowestLexicographicPeerID(t *testing.T) {
	p1 := mustPeerID(t, "12D3KooWQf89meYvGVnR8aqW3w5ixuY9FB4CvA8Hqv7BqkKJfmg1")
	p2 := mustPeerID(t, "12D3KooWQnD5Wofx8WGfFWD3ssuZrXRAjnf8Y3Wg6vVHvB4x8S7d")
	p3 := mustPeerID(t, "12D3KooWSEi43o95aMw16J9J4rq2e8Vr1H8QfA6oZxwA7tD9u9cD")

	host := ElectHost(p3, []peer.ID{p1, p2})
	if host != p1 {
		t.Fatalf("expected host %s, got %s", p1, host)
	}
}

func TestIsHost_ReturnsTrueOnlyForLeader(t *testing.T) {
	leader := mustPeerID(t, "12D3KooWQf89meYvGVnR8aqW3w5ixuY9FB4CvA8Hqv7BqkKJfmg1")
	other := mustPeerID(t, "12D3KooWQnD5Wofx8WGfFWD3ssuZrXRAjnf8Y3Wg6vVHvB4x8S7d")

	if !IsHost(leader, []peer.ID{other}) {
		t.Fatalf("expected leader to be host")
	}
	if IsHost(other, []peer.ID{leader}) {
		t.Fatalf("expected non-leader to not be host")
	}
}
