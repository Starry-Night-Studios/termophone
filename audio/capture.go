package audio

import (
	"github.com/gen2brain/malgo"
)

type Capturer struct {
	device *malgo.Device
}

func NewCapturer(ctx *malgo.AllocatedContext, rawCh chan<- []byte) (*Capturer, error) {
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

	var capBuf []byte

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, inputSamples []byte, _ uint32) {
			if len(inputSamples) == 0 {
				return
			}

			// Append new OS audio data to our temporary capture buffer
			capBuf = append(capBuf, inputSamples...)

			// Slice it into perfect 10ms (FrameBytes) chunks
			for len(capBuf) >= FrameBytes {
				frame := make([]byte, FrameBytes)
				copy(frame, capBuf[:FrameBytes])

				// Re-use backing array memory to stop GC stalls (cracking)
				copy(capBuf, capBuf[FrameBytes:])
				capBuf = capBuf[:len(capBuf)-FrameBytes]

				select {
				case rawCh <- frame:
				default:
					// Drop frame if pipeline is busy (prevents lag building up)
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
