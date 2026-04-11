package net

import (
	"context"

	videonet "termophone/video"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const VideoProtocolID = videonet.ProtocolID

type Quality = videonet.Quality

var Qualities = videonet.Qualities

func StartScreenShare(ctx context.Context, h host.Host, peerID peer.ID, quality string) error {
	return videonet.StartScreenShare(ctx, h, peerID, quality)
}

func ReceiveScreenShare(ctx context.Context, s network.Stream) error {
	return videonet.ReceiveScreenShare(ctx, s)
}
