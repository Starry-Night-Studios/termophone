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
	// Add 1 to ensure the ring buffer size actually creates the requested delay.
	delayFrames := (p.aecDelay / 10) + 1
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

			// Pass aecOut directly! No heap allocations.
			if isSilent := p.denoiser.Process(aecOut); isSilent {
				silentFrames++
				// Return the unprocessed buffer to the zero-allocation pool
				select {
				case p.freePool <- frame:
				default:
				}
				// Skip the Opus encoder and network send completely!
				continue
			}

			// Cast clean []int16 back to []byte
			byteLen := len(aecOut) * 2
			cleanBytes := unsafe.Slice((*byte)(unsafe.Pointer(&aecOut[0])), byteLen)

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

func RecvPipeline(recvCh <-chan []byte, rb *RingBuffer, codec *Codec) {
	rb.Reset()

	logTicker := time.NewTicker(time.Second * 2)
	defer logTicker.Stop()
	var framesReceived, framesDecoded uint64
	var maxSeq uint16
	var hasSeq bool

	// 100ms without packets means the sender stopped speaking (silence suppression)
	talkSpurtTicker := time.NewTicker(time.Millisecond * 100)
	defer talkSpurtTicker.Stop()

	for {
		select {
		case frame, ok := <-recvCh:
			if !ok {
				return
			}
			talkSpurtTicker.Reset(time.Millisecond * 100)

			if len(frame) < 2 {
				continue
			}

			// Sequence tracking
			seq := binary.LittleEndian.Uint16(frame[0:2])
			payload := frame[2:]
			if len(payload) == 0 {
				continue
			}

			if hasSeq {
				diff := int16(seq - maxSeq)

				// Stale packet (arrived too late)
				if diff < -5 && diff > -1000 {
					continue
				}

				// SEQUENCE-BASED PLC: Accurately synthesize missing packets
				if diff > 1 && diff < 10 {
					for i := int16(0); i < diff-1; i++ {
						if plcFrame := codec.DecodePLC(); plcFrame != nil {
							rb.Write(plcFrame)
						}
					}
				}

				if diff > 0 {
					maxSeq = seq
				}
			} else {
				hasSeq = true
				maxSeq = seq
			}

			framesReceived++

			// Decompress
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

		case <-talkSpurtTicker.C:
			// Sender stopped speaking.
			// Reset the buffer so the NEXT sentence gets properly pre-buffered!
			rb.Reset()
			hasSeq = false // Reset sequence tracking for the new talk spurt

		case <-logTicker.C:
			if framesReceived > 0 {
				log.Printf("[MAC/RECV Debug] Receiving... Opus Packets:%d | Successfully Decoded/Played:%d", framesReceived, framesDecoded)
			}
		}
	}
}
