package audio

/*
#cgo pkg-config: rnnoise
#include <rnnoise.h>
#include <stdlib.h>
*/
import "C"

// DenoiseState wraps the RNNoise C state
type DenoiseState struct {
	state   *C.DenoiseState
	inFloat []C.float
}

// NewDenoiseState creates a new instance of the rnnoise model
func NewDenoiseState() *DenoiseState {
	return &DenoiseState{
		state:   C.rnnoise_create(nil),
		inFloat: make([]C.float, FrameSamples),
	}
}

// Process cleans background noise and keyboard clicks from an audio frame.
// Modifies the pcm slice in place. Returns true if perfectly silent.
func (d *DenoiseState) Process(pcm []int16) bool {
	if d.state == nil || len(pcm) != FrameSamples {
		return false
	}

	isSilent := true
	for _, s := range pcm {
		if s != 0 {
			isSilent = false
			break
		}
	}

	// Skip expensive neural net processing if the mic frame is perfectly silent
	if isSilent {
		return true
	}

	for i, s := range pcm {
		d.inFloat[i] = C.float(s)
	}

	// Run RNNoise neural network model
	C.rnnoise_process_frame(d.state, (*C.float)(&d.inFloat[0]), (*C.float)(&d.inFloat[0]))

	// Convert the cleaned floats back to int16
	for i := 0; i < FrameSamples; i++ {
		val := d.inFloat[i]
		if val > 32767.0 {
			pcm[i] = 32767
		} else if val < -32768.0 {
			pcm[i] = -32768
		} else {
			pcm[i] = int16(val)
		}
	}
	return false
}

// Close frees the RNNoise C memory
func (d *DenoiseState) Close() {
	if d.state != nil {
		C.rnnoise_destroy(d.state)
		d.state = nil
	}
}
