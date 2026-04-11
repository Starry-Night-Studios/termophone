package audio

import (
	"context"
	"encoding/binary"
	"log"
	"time"
	"unsafe"
)

const (
	targetFill = FrameBytes * 8  // ~80ms target latency
	maxFill    = FrameBytes * 32 // ~320ms before drift correction
)

// Pipeline sits between capture and network.
type Pipeline struct {
	rawCh    <-chan []byte
	sendCh   chan<- []byte
	codec    *Codec
	aec      *EchoCanceller
	denoiser *DenoiseState
	rb       *RingBuffer
	aecDelay int
	freePool chan []byte
}

func NewPipeline(rawCh <-chan []byte, sendCh chan<- []byte, rb *RingBuffer, aecDelay int, freePool chan []byte) *Pipeline {
	// 480 frames = 10ms. 12000 tail = 250ms echo memory
	echoCanceller, err := NewEchoCanceller(FrameSamples, 12000, SampleRate)
	if err != nil {
		panic(err)
	}

	codec, err := NewCodec()
	if err != nil {
		panic(err)
	}

	return &Pipeline{
		rawCh:    rawCh,
		sendCh:   sendCh,
		codec:    codec,
		aec:      echoCanceller,
		denoiser: NewDenoiseState(),
		rb:       rb,
		aecDelay: aecDelay,
		freePool: freePool,
	}
}

func (p *Pipeline) Run(ctx context.Context) {
	defer p.aec.Close()
	defer p.denoiser.Close()
	refBuf := make([]byte, FrameBytes)

	logTicker := time.NewTicker(time.Second * 2)
	defer logTicker.Stop()
	var framesCaptured, silentFrames, framesSent uint64

	// --- Hardware Latency Delay Line Setup ---
	// p.aecDelay is config.AECTrimOffsetMs. Since each frame is 10ms, we need delayFrames.
	// Ensure delayFrames is at least 1 to avoid popping from empty
	delayFrames := p.aecDelay / 10
	if delayFrames < 1 {
		delayFrames = 1
	}

	// Create a queue to hold our delayed reference frames
	refRing := make([][]int16, delayFrames)

	// Pre-fill the delay line with digital silence (zeros)
	for i := range refRing {
		refRing[i] = make([]int16, FrameSamples)
	}
	refHead := 0
	// ----------------------------------------------

	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-p.rawCh:
			framesCaptured++

			// Get the CURRENT speaker audio and the CURRENT mic audio
			p.rb.Peek(refBuf)
			mic16 := unsafe.Slice((*int16)(unsafe.Pointer(&frame[0])), len(frame)/2)
			ref16 := unsafe.Slice((*int16)(unsafe.Pointer(&refBuf[0])), len(refBuf)/2)

			// Store latest reference frame in the circular buffer
			copy(refRing[refHead], ref16)

			// Extract the legitimately delayed frame from the back of the queue
			delayedRef16 := refRing[(refHead+1)%delayFrames]
			refHead = (refHead + 1) % delayFrames

			// Process echo cancellation
			aecOut := p.aec.Process(mic16, delayedRef16)
			clean16 := make([]int16, len(aecOut))
			copy(clean16, aecOut)

			// Fast silence check and Denoise step wrapped into one
			if isSilent := p.denoiser.Process(clean16); isSilent {
				silentFrames++
			}

			// Cast clean []int16 back to []byte
			byteLen := len(clean16) * 2
			cleanBytes := unsafe.Slice((*byte)(unsafe.Pointer(&clean16[0])), byteLen)

			// Compress clean audio
			encodedFrame := p.codec.Encode(cleanBytes)

			// Return the now-processed capture frame to the zero-allocation free list
			select {
			case p.freePool <- frame:
			default:
			}

			if encodedFrame == nil {
				continue
			}

			select {
			case p.sendCh <- encodedFrame:
				framesSent++
			default:
			}
		case <-logTicker.C:
			if framesCaptured > 0 {
				log.Printf("[MAC/SENDER Debug] Capturing... Mic Frames:%d (Silent:%d) | Sent to Network:%d", framesCaptured, silentFrames, framesSent)
			}
		}
	}
}

// RecvPipeline sits between network and playback.
// recvCh → decode → ring buffer
func RecvPipeline(recvCh <-chan []byte, rb *RingBuffer, codec *Codec) {
	logTicker := time.NewTicker(time.Second * 2)
	defer logTicker.Stop()
	var framesReceived, framesDecoded uint64
	var maxSeq uint16
	var hasSeq bool

	// Ticker for network gap detection (every 12ms allows 2ms leeway over a 10ms frame rate)
	gapTicker := time.NewTicker(time.Millisecond * 12)
	defer gapTicker.Stop()

	for {
		select {
		case frame, ok := <-recvCh:
			if !ok {
				return
			}
			// Reset gap timer because we just got a real packet!
			gapTicker.Reset(time.Millisecond * 12)

			if len(frame) < 2 {
				continue
			}

			// Sequence tracking
			seq := binary.LittleEndian.Uint16(frame[0:2])
			payload := frame[2:]
			if len(payload) == 0 {
				// Empty Opus packet can happen with malformed frames; skip quietly.
				continue
			}

			if hasSeq {
				diff := int16(seq - maxSeq)
				// Allow a small reorder window (±5 sequence numbers).
				// Anything significantly older (<-5) in a rapid burst is dropped to shed latency.
				if diff < -5 && diff > -1000 {
					// Stale packet (arrived too late), drop it to shed latency
					continue
				}
				if diff > 0 {
					maxSeq = seq
				}
			} else {
				hasSeq = true
				maxSeq = seq
			}

			framesReceived++

			// Decompress!
			decodedFrame := codec.Decode(payload)
			if decodedFrame == nil {
				continue
			}
			framesDecoded++

			if rb.Fill() > maxFill {
				rb.DropToTarget(targetFill)
			}

			if !rb.Write(decodedFrame) {
				log.Println("Audio receive ring buffer full, dropped frame!")
			}

		case <-gapTicker.C:
			// Network Gap detected! The other side is lagging or a packet dropped.
			// Trigger Opus Packet Loss Concealment (PLC)

			// Synthesize 10ms extrapolated audio from the decoder state.
			plcFrame := codec.DecodePLC()
			if plcFrame != nil {
				if !rb.Write(plcFrame) {
					log.Println("Audio receive ring buffer full, dropped PLC frame!")
				}
			}

		case <-logTicker.C:
			if framesReceived > 0 {
				log.Printf("[MAC/RECV Debug] Receiving... Opus Packets:%d | Successfully Decoded/Played:%d", framesReceived, framesDecoded)
			}
		}
	}
}
