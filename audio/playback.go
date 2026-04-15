package audio

import (
	"sync"

	"github.com/gen2brain/malgo"
)

type RingBuffer struct {
	buf          []byte
	r, w         int
	size         int
	mu           sync.Mutex
	prebuffering bool
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{buf: make([]byte, size), size: size, prebuffering: true}
}

func (rb *RingBuffer) Write(p []byte) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.size-(rb.w-rb.r) < len(p) {
		return false
	}
	wPos := rb.w % rb.size
	n := copy(rb.buf[wPos:], p)
	if n < len(p) {
		copy(rb.buf[0:], p[n:])
	}
	rb.w += len(p)
	return true
}

func (rb *RingBuffer) readAt(dst []byte, pos int) {
	rPos := pos % rb.size
	n := copy(dst, rb.buf[rPos:])
	if n < len(dst) {
		copy(dst[n:], rb.buf[0:])
	}
}

func (rb *RingBuffer) Read(p []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	avail := rb.w - rb.r
	n := len(p)

	// Prevent stuttering by ensuring we have a cushion of ~40ms (4 frames minimum)
	if rb.prebuffering {
		if avail < FrameBytes*4 {
			for i := 0; i < n; i++ {
				p[i] = 0
			}
			return
		}
		rb.prebuffering = false
	}

	if avail < n {
		// Output pure silence to avoid waveform cliffs/buzzing.
		for i := 0; i < n; i++ {
			p[i] = 0
		}
		// Do NOT set rb.prebuffering = true here.
		// We just wait patiently for the late packet to arrive.
		return
	}

	rb.readAt(p, rb.r)
	rb.r += n
}

// Peek copies the next n bytes from the ring buffer without moving the read pointer
func (rb *RingBuffer) Peek(p []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	avail := rb.w - rb.r
	n := len(p)
	if avail < n {
		rb.readAt(p[:avail], rb.r)
		for i := avail; i < n; i++ {
			p[i] = 0
		}
		return
	}
	rb.readAt(p, rb.r)
}

func (rb *RingBuffer) Fill() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.w - rb.r
}

// DropToTarget discards bytes so that Fill() == target, used for drift correction.
func (rb *RingBuffer) DropToTarget(target int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	avail := rb.w - rb.r
	if avail > target {
		rb.r += avail - target
		rb.prebuffering = false
	}
}

// Reset flushes all buffered audio and re-enables prebuffering.
// Call this when a new peer connection starts so stale audio from
// a previous session (or runaway PLC frames) doesn't pollute the
// fresh stream.
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.r = 0
	rb.w = 0
	rb.prebuffering = true
}

type Player struct {
	rb     *RingBuffer
	device *malgo.Device
}

func NewPlayer(ctx *malgo.AllocatedContext, rb *RingBuffer) (*Player, error) {
	p := &Player{rb: rb}

	cfg := malgo.DefaultDeviceConfig(malgo.Playback)
	cfg.Playback.Format = malgo.FormatS16
	cfg.Playback.Channels = Channels
	cfg.SampleRate = SampleRate

	// Removed WASAPI/ALSA flags from playback configuration

	callbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, _ []byte, _ uint32) {
			rb.Read(outputSamples)
		},
	}

	device, err := malgo.InitDevice(ctx.Context, cfg, callbacks)
	if err != nil {
		return nil, err
	}

	p.device = device
	return p, nil
}

func (p *Player) Start() error { return p.device.Start() }
func (p *Player) Stop()        { p.device.Stop() }
func (p *Player) Uninit()      { p.device.Uninit() }
