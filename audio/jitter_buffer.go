package audio

import "sync"

// JitterBuffer is a fixed-size, sequence-aware circular buffer for Opus packets.
type JitterBuffer struct {
	mu sync.Mutex

	buffer  [][]byte
	seqMap  []uint16
	present []bool

	size        uint16
	prefill     int
	playSeq     uint16
	initialized bool
	buffering   bool
	buffered    int
}

func NewJitterBuffer(size uint16, prefill int) *JitterBuffer {
	if size == 0 {
		size = 8
	}
	if prefill < 1 {
		prefill = 1
	}
	if prefill > int(size) {
		prefill = int(size)
	}

	return &JitterBuffer{
		buffer:    make([][]byte, size),
		seqMap:    make([]uint16, size),
		present:   make([]bool, size),
		size:      size,
		prefill:   prefill,
		buffering: true,
	}
}

// Push inserts an encoded packet by sequence number.
func (jb *JitterBuffer) Push(seq uint16, payload []byte) {
	if len(payload) == 0 {
		return
	}

	jb.mu.Lock()
	defer jb.mu.Unlock()

	if !jb.initialized {
		jb.initialized = true
		jb.buffering = true
		jb.playSeq = seq - uint16(jb.prefill-1)
	}

	diff := int16(seq - jb.playSeq)
	if diff < 0 {
		// Packet is older than current playout sequence.
		return
	}

	if diff >= int16(jb.size) {
		// Packet is far ahead of playout window; jump the window.
		jb.playSeq = seq - uint16(jb.prefill-1)
		jb.resetLocked()
		jb.buffering = true
	}

	idx := seq % jb.size
	if jb.present[idx] && jb.seqMap[idx] == seq {
		return
	}

	buf := make([]byte, len(payload))
	copy(buf, payload)

	if !jb.present[idx] {
		jb.buffered++
	}
	jb.buffer[idx] = buf
	jb.seqMap[idx] = seq
	jb.present[idx] = true
}

// Pop returns the next in-order payload if available.
func (jb *JitterBuffer) Pop() ([]byte, bool) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if !jb.initialized {
		return nil, false
	}

	if jb.buffering {
		if jb.buffered < jb.prefill {
			return nil, false
		}
		jb.buffering = false
	}

	idx := jb.playSeq % jb.size
	if jb.present[idx] && jb.seqMap[idx] == jb.playSeq {
		out := jb.buffer[idx]
		jb.buffer[idx] = nil
		jb.present[idx] = false
		jb.buffered--
		jb.playSeq++
		return out, true
	}

	jb.playSeq++
	return nil, false
}

func (jb *JitterBuffer) resetLocked() {
	for i := range jb.buffer {
		jb.buffer[i] = nil
		jb.present[i] = false
	}
	jb.buffered = 0
}
