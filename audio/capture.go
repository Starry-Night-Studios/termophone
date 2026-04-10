package audio

import (
	"github.com/gen2brain/malgo"
)

type Capturer struct {
	device *malgo.Device
}

func NewCapturer(ctx *malgo.AllocatedContext, rawCh chan<- []byte, freePool chan []byte) (*Capturer, error) {
	c := &Capturer{}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = Channels
	cfg.SampleRate = SampleRate

	// Bypasses the Windows Audio Engine (WASAPI) software mixers and enhancements
	cfg.Wasapi.NoAutoConvertSRC = 1
	cfg.Wasapi.NoDefaultQualitySRC = 1
	cfg.Wasapi.NoAutoStreamRouting = 1
	cfg.Alsa.NoMMap = 1

	// Pre-allocated fixed-size ring buffer for incoming OS audio data
	// 100ms capacity (10 frames) is plenty for the capture side
	capRb := NewRingBuffer(FrameBytes * 10)
	capRb.prebuffering = false // Capture doesn't need prebuffering logic

	// Single, pre-allocated trash array for when the pipeline falls critically behind
	trashFrame := make([]byte, FrameBytes)

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, inputSamples []byte, _ uint32) {
			if len(inputSamples) == 0 {
				return
			}

			// Push raw hardware audio into our allocation-free ring buffer
			capRb.Write(inputSamples)

			// Slice it into perfect 10ms (FrameBytes) chunks
			for capRb.Fill() >= FrameBytes {
				var frame []byte
				select {
				case frame = <-freePool:
				default:
					// Pool exhausted! Pipeline is backed up. Drop newest frame.
					capRb.Read(trashFrame)
					continue
				}

				// Read directly into our pre-allocated, GC-immune byte slice
				capRb.Read(frame)

				select {
				case rawCh <- frame:
				default:
					// Drop frame if pipeline is busy (prevents lag building up)
					// Return to pool so we don't leak it!
					select {
					case freePool <- frame:
					default:
					}
				}
			}
		},
	}

	device, err := malgo.InitDevice(ctx.Context, cfg, callbacks)
	if err != nil {
		return nil, err
	}

	c.device = device
	return c, nil
}

func (c *Capturer) Start() error { return c.device.Start() }
func (c *Capturer) Stop()        { c.device.Stop() }
func (c *Capturer) Uninit()      { c.device.Uninit() }
