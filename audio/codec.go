package audio

import (
	"log"
	"unsafe"

	"github.com/hraban/opus"
)

type Codec struct {
	encoder *opus.Encoder
	decoder *opus.Decoder
	encBuf  []byte
	decBuf  []int16
}

func NewCodec() *Codec {
	enc, err := opus.NewEncoder(SampleRate, Channels, opus.AppVoIP)
	if err != nil {
		log.Fatalf("failed to create opus encoder: %v", err)
	}

	dec, err := opus.NewDecoder(SampleRate, Channels)
	if err != nil {
		log.Fatalf("failed to create opus decoder: %v", err)
	}

	return &Codec{
		encoder: enc,
		decoder: dec,
		encBuf:  make([]byte, 1000),
		decBuf:  make([]int16, 5760),
	}
}

// Encode compresses raw PCM data to Opus an payload
func (c *Codec) Encode(pcm []byte) []byte {
	// Cast []byte to []int16 for the Opus encoder
	pcm16 := unsafe.Slice((*int16)(unsafe.Pointer(&pcm[0])), len(pcm)/2)

	n, err := c.encoder.Encode(pcm16, c.encBuf)
	if err != nil {
		log.Println("opus encode error:", err)
		return nil
	}

	out := make([]byte, n)
	copy(out, c.encBuf[:n])
	return out
}

// Decode decompresses an Opus payload back to raw PCM data
func (c *Codec) Decode(data []byte) []byte {
	n, err := c.decoder.Decode(data, c.decBuf)
	if err != nil {
		log.Println("opus decode error:", err)
		return nil
	}

	// Cast decoded []int16 back to []byte to push back onto the audio ring buffer
	byteLen := n * 2 // n is frames, 2 bytes per int16 sample
	out := make([]byte, byteLen)
	decBytes := unsafe.Slice((*byte)(unsafe.Pointer(&c.decBuf[0])), byteLen)

	copy(out, decBytes)
	return out
}
