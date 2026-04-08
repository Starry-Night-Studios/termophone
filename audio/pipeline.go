package audio

import (
	"context"
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
}

func NewPipeline(rawCh <-chan []byte, sendCh chan<- []byte, rb *RingBuffer) *Pipeline {
	// 480 frames = 10ms. 12000 tail = 250ms echo memory
	echoCanceller, err := NewEchoCanceller(FrameSamples, 12000, SampleRate)
	if err != nil {
		panic(err)
	}

	return &Pipeline{
		rawCh:    rawCh,
		sendCh:   sendCh,
		codec:    NewCodec(),
		aec:      echoCanceller,
		denoiser: NewDenoiseState(),
		rb:       rb,
	}
}

func (p *Pipeline) Run(ctx context.Context) {
	defer p.aec.Close()
	defer p.denoiser.Close()
	refBuf := make([]byte, FrameBytes)

	logTicker := time.NewTicker(time.Second * 2)
	defer logTicker.Stop()
	var framesCaptured, silentFrames, framesSent uint64

	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-p.rawCh:
			framesCaptured++

			// 2. AEC
			// Peek at the audio that is currently about to play on the speakers
			p.rb.Peek(refBuf)

			// Cast the []byte slices to []int16 for Speex
			mic16 := unsafe.Slice((*int16)(unsafe.Pointer(&frame[0])), len(frame)/2)
			ref16 := unsafe.Slice((*int16)(unsafe.Pointer(&refBuf[0])), len(refBuf)/2)

			// Process echo cancellation
			clean16 := p.aec.Process(mic16, ref16)

			// Fast silence check and Denoise step wrapped into one
			if isSilent := p.denoiser.Process(clean16); isSilent {
				silentFrames++
			}

			// Cast clean []int16 back to []byte
			byteLen := len(clean16) * 2
			cleanBytes := unsafe.Slice((*byte)(unsafe.Pointer(&clean16[0])), byteLen)

			// 3. Compress clean audio
			encodedFrame := p.codec.Encode(cleanBytes)
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

	for {
		select {
		case frame, ok := <-recvCh:
			if !ok {
				return
			}
			framesReceived++

			// Decompress!
			decodedFrame := codec.Decode(frame)
			if decodedFrame == nil {
				continue
			}
			framesDecoded++

			if rb.Fill() > maxFill {
				rb.DropToTarget(targetFill)
			}

			rb.Write(decodedFrame)
		case <-logTicker.C:
			if framesReceived > 0 {
				log.Printf("[MAC/RECV Debug] Receiving... Opus Packets:%d | Successfully Decoded/Played:%d", framesReceived, framesDecoded)
			}
		}
	}
}
