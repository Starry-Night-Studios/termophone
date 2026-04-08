package audio

/*
#cgo pkg-config: speexdsp
#include <speex/speex_echo.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// EchoCanceller wraps the SpeexDSP Echo Cancellation state
type EchoCanceller struct {
	state    *C.SpeexEchoState
	outClean []int16
}

// NewEchoCanceller creates a new acoustic echo canceller
// frameSize is the size of the frames in samples (e.g., 480 for 10ms at 48kHz)
// tailLength is the window in samples that AEC looks for echoes (e.g., 4800 for 100ms)
func NewEchoCanceller(frameSize, tailLength int, sampleRate int) (*EchoCanceller, error) {
	state := C.speex_echo_state_init(C.int(frameSize), C.int(tailLength))
	if state == nil {
		return nil, fmt.Errorf("failed to initialize speex AEC")
	}

	// Set the sample rate for the echo canceller
	sr := C.spx_int32_t(sampleRate)
	C.speex_echo_ctl(state, C.SPEEX_ECHO_SET_SAMPLING_RATE, unsafe.Pointer(&sr))

	return &EchoCanceller{
		state:    state,
		outClean: make([]int16, frameSize),
	}, nil
}

// Process removes echo from the microphone input signal
// micRaw: The audio just captured from the local microphone
// speakerRef: The audio that was just played out of the local speakers
func (aec *EchoCanceller) Process(micRaw []int16, speakerRef []int16) []int16 {
	if aec.state == nil || len(micRaw) != len(aec.outClean) {
		return micRaw
	}

	// Ensure we have correct pointer casts for Speex
	C.speex_echo_cancellation(
		aec.state,
		(*C.spx_int16_t)(unsafe.Pointer(&micRaw[0])),
		(*C.spx_int16_t)(unsafe.Pointer(&speakerRef[0])),
		(*C.spx_int16_t)(unsafe.Pointer(&aec.outClean[0])),
	)

	return aec.outClean
}

// Close frees the Speex C memory
func (aec *EchoCanceller) Close() {
	if aec.state != nil {
		C.speex_echo_state_destroy(aec.state)
		aec.state = nil
	}
}
