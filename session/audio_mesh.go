package session

import (
	"context"
	"encoding/binary"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"termophone/audio"
	vnet "termophone/net"

	"github.com/gen2brain/malgo"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type remotePeerAudio struct {
	sendCh chan []byte
	recvCh chan []byte
	codec  *audio.Codec
	jitter *audio.JitterBuffer
	done   chan struct{}

	mu           sync.RWMutex
	lastFrame    []int16
	lastPacketAt time.Time

	closeOnce sync.Once
}

type AudioMeshSession struct {
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once

	h     host.Host
	muted *atomic.Bool

	onPeerConnected       func(id, name string)
	onSessionDisconnected func()

	rawCh    chan []byte
	sendCh   chan []byte
	freePool chan []byte
	rb       *audio.RingBuffer

	mctx     *malgo.AllocatedContext
	capturer *audio.Capturer
	player   *audio.Player

	peersMu sync.RWMutex
	peers   map[peer.ID]*remotePeerAudio
}

func NewAudioMeshSession(
	parent context.Context,
	h host.Host,
	muted *atomic.Bool,
	onPeerConnected func(id, name string),
	onSessionDisconnected func(),
	aecDelayMs int,
) (*AudioMeshSession, error) {
	ctx, cancel := context.WithCancel(parent)

	s := &AudioMeshSession{
		ctx:                   ctx,
		cancel:                cancel,
		h:                     h,
		muted:                 muted,
		onPeerConnected:       onPeerConnected,
		onSessionDisconnected: onSessionDisconnected,
		rawCh:                 make(chan []byte, 8),
		sendCh:                make(chan []byte, 8),
		freePool:              make(chan []byte, 32),
		rb:                    audio.NewRingBuffer(1024 * 64),
		peers:                 make(map[peer.ID]*remotePeerAudio),
	}

	for i := 0; i < 32; i++ {
		s.freePool <- make([]byte, audio.FrameBytes)
	}

	var err error
	s.mctx, err = malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		log.Print(msg)
	})
	if err != nil {
		cancel()
		return nil, err
	}

	s.capturer, err = audio.NewCapturer(s.mctx, s.rawCh, s.freePool)
	if err != nil {
		s.Close()
		return nil, err
	}

	s.player, err = audio.NewPlayer(s.mctx, s.rb)
	if err != nil {
		s.Close()
		return nil, err
	}

	go audio.NewPipeline(s.rawCh, s.sendCh, s.rb, aecDelayMs, s.freePool).Run(s.ctx)
	go s.broadcastLoop()
	go s.mixerLoop()

	if err := s.capturer.Start(); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.player.Start(); err != nil {
		s.Close()
		return nil, err
	}

	go func() {
		<-s.ctx.Done()
		s.Close()
	}()

	return s, nil
}

func (s *AudioMeshSession) Close() {
	s.closeOnce.Do(func() {
		s.cancel()

		s.peersMu.Lock()
		for _, rp := range s.peers {
			rp.closeOnce.Do(func() {
				close(rp.sendCh)
				close(rp.done)
			})
		}
		s.peers = map[peer.ID]*remotePeerAudio{}
		s.peersMu.Unlock()

		if s.capturer != nil {
			s.capturer.Stop()
			s.capturer.Uninit()
			s.capturer = nil
		}
		if s.player != nil {
			s.player.Stop()
			s.player.Uninit()
			s.player = nil
		}
		if s.mctx != nil {
			s.mctx.Free()
			s.mctx = nil
		}
	})
}

func (s *AudioMeshSession) AddStream(stream network.Stream) {
	pid := stream.Conn().RemotePeer()

	rp := &remotePeerAudio{
		sendCh: make(chan []byte, 8),
		recvCh: make(chan []byte, 16),
		jitter: audio.NewJitterBuffer(8, 3),
		done:   make(chan struct{}),
	}
	codec, err := audio.NewCodec()
	if err != nil {
		log.Printf("Failed to create codec for peer %s: %v", pid, err)
		stream.Reset()
		return
	}
	rp.codec = codec

	s.peersMu.Lock()
	if old, ok := s.peers[pid]; ok {
		old.closeOnce.Do(func() {
			close(old.sendCh)
			close(old.done)
		})
	}
	s.peers[pid] = rp
	s.peersMu.Unlock()

	if s.onPeerConnected != nil {
		s.onPeerConnected(pid.String(), s.displayName(pid))
	}

	go func() {
		defer close(rp.recvCh)
		vnet.Reader(stream, rp.recvCh)
		s.removePeer(pid)
	}()

	go func() {
		vnet.Writer(stream, rp.sendCh)
		s.removePeer(pid)
	}()

	go func() {
		for frame := range rp.recvCh {
			if len(frame) < 2 {
				continue
			}
			seq := binary.LittleEndian.Uint16(frame[0:2])
			rp.jitter.Push(seq, frame[2:])
			rp.mu.Lock()
			rp.lastPacketAt = time.Now()
			rp.mu.Unlock()
		}
	}()

	go s.peerPlayoutLoop(rp)
}

func (s *AudioMeshSession) removePeer(pid peer.ID) {
	s.peersMu.Lock()
	rp, ok := s.peers[pid]
	if ok {
		delete(s.peers, pid)
	}
	empty := len(s.peers) == 0
	s.peersMu.Unlock()

	if ok {
		rp.closeOnce.Do(func() {
			close(rp.sendCh)
			close(rp.done)
		})
	}

	if empty && s.onSessionDisconnected != nil {
		s.onSessionDisconnected()
	}
}

func (s *AudioMeshSession) peerPlayoutLoop(rp *remotePeerAudio) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	silence := make([]int16, audio.FrameSamples)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-rp.done:
			return
		case <-ticker.C:
			payload, ok := rp.jitter.Pop()
			frame := silence
			if ok {
				decoded := rp.codec.Decode(payload)
				if decoded != nil {
					frame = bytesToPCM(decoded)
				}
			}

			rp.mu.Lock()
			rp.lastFrame = frame
			rp.mu.Unlock()
		}
	}
}

func (s *AudioMeshSession) broadcastLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case frame := <-s.sendCh:
			if s.muted.Load() {
				continue
			}
			for _, rp := range s.peerSnapshot() {
				select {
				case rp.sendCh <- frame:
				default:
					// Backpressured peers drop a frame instead of blocking all peers.
				}
			}
		}
	}
}

func (s *AudioMeshSession) mixerLoop() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			frames := s.activeFrames(150 * time.Millisecond)
			if len(frames) == 0 {
				continue
			}

			mixed := audio.MixAudio(frames, audio.FrameSamples)
			pcm := pcmToBytes(mixed)
			if s.rb.Fill() > audio.FrameBytes*32 {
				s.rb.DropToTarget(audio.FrameBytes * 8)
			}
			if !s.rb.Write(pcm) {
				s.rb.DropToTarget(audio.FrameBytes * 8)
				_ = s.rb.Write(pcm)
			}
		}
	}
}

func (s *AudioMeshSession) peerSnapshot() []*remotePeerAudio {
	s.peersMu.RLock()
	out := make([]*remotePeerAudio, 0, len(s.peers))
	for _, rp := range s.peers {
		out = append(out, rp)
	}
	s.peersMu.RUnlock()
	return out
}

func (s *AudioMeshSession) activeFrames(maxAge time.Duration) [][]int16 {
	now := time.Now()
	frames := make([][]int16, 0)
	for _, rp := range s.peerSnapshot() {
		rp.mu.RLock()
		fresh := !rp.lastPacketAt.IsZero() && now.Sub(rp.lastPacketAt) <= maxAge
		if fresh && len(rp.lastFrame) > 0 {
			buf := make([]int16, len(rp.lastFrame))
			copy(buf, rp.lastFrame)
			frames = append(frames, buf)
		}
		rp.mu.RUnlock()
	}
	return frames
}

func (s *AudioMeshSession) displayName(id peer.ID) string {
	agent, err := s.h.Peerstore().Get(id, "AgentVersion")
	name := id.String()
	if err == nil && agent != nil {
		if str, ok := agent.(string); ok && strings.HasPrefix(str, "termophone/") {
			return strings.TrimPrefix(str, "termophone/")
		}
	}
	if len(name) > 12 {
		return name[len(name)-8:]
	}
	return name
}

func bytesToPCM(frame []byte) []int16 {
	samples := len(frame) / 2
	pcm := make([]int16, samples)
	for i := 0; i < samples; i++ {
		pcm[i] = int16(binary.LittleEndian.Uint16(frame[i*2 : i*2+2]))
	}
	return pcm
}

func pcmToBytes(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, v := range samples {
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(v))
	}
	return out
}
