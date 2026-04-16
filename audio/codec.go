package audio

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/hraban/opus"
)

type Codec struct {
	encoder *opus.Encoder
	decoder *opus.Decoder
	encBuf  []byte
	decBuf  []int16

	decodeErrSuppressed int
	lastDecodeErrLog    time.Time
	plcErrSuppressed    int
	lastPLCErrLog       time.Time
}

func NewCodec() (*Codec, error) {

	enc, err := opus.NewEncoder(SampleRate, Channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %v", err)
	}

	// Set Bitrate to 32kbps
	if err := enc.SetBitrate(32000); err != nil {
		log.Printf("opus: failed to set bitrate: %v", err)
	}
	// Enable In-Band FEC
	if err := enc.SetInBandFEC(true); err != nil {
		log.Printf("opus: failed to enable FEC: %v", err)
	}
	// Set expected Packet Loss Percentage to 15
	if err := enc.SetPacketLossPerc(15); err != nil {
		log.Printf("opus: failed to set packet loss perc: %v", err)
	}
	// Enable DTX
	if err := enc.SetDTX(true); err != nil {
		log.Printf("opus: failed to enable DTX: %v", err)
	}

	dec, err := opus.NewDecoder(SampleRate, Channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus decoder: %v", err)
	}

	return &Codec{
		encoder: enc,
		decoder: dec,
		encBuf:  make([]byte, 1000),
		decBuf:  make([]int16, 5760),
	}, nil
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

	// Return a COPY so the network layer doesn't see its data change mid-send
	out := make([]byte, n)
	copy(out, c.encBuf[:n])
	return out
}

// Decode decompresses an Opus payload back to raw PCM data
func (c *Codec) Decode(data []byte) []byte {
	if len(data) == 0 {
		// Empty payload is not valid for Decode; callers should use DecodePLC.
		return nil
	}

	n, err := c.decoder.Decode(data, c.decBuf)
	if err != nil {
		now := time.Now()
		if c.lastDecodeErrLog.IsZero() || now.Sub(c.lastDecodeErrLog) >= 3*time.Second {
			if c.decodeErrSuppressed > 0 {
				log.Printf("opus decode error (non-fatal, payload=%dB): %v (suppressed %d similar errors)", len(data), err, c.decodeErrSuppressed)
				c.decodeErrSuppressed = 0
			} else {
				log.Printf("opus decode error (non-fatal, payload=%dB): %v", len(data), err)
			}
			c.lastDecodeErrLog = now
		} else {
			c.decodeErrSuppressed++
		}
		return nil
	}

	// Cast decoded []int16 back to []byte to push back onto the audio ring buffer
	byteLen := n * 2 // n is frames, 2 bytes per int16 sample
	return unsafe.Slice((*byte)(unsafe.Pointer(&c.decBuf[0])), byteLen)
}

// DecodePLC synthesizes a frame during packet gaps using Opus packet-loss concealment.
func (c *Codec) DecodePLC() []byte {
	err := c.decoder.DecodePLC(c.decBuf)
	if err != nil {
		now := time.Now()
		if c.lastPLCErrLog.IsZero() || now.Sub(c.lastPLCErrLog) >= 3*time.Second {
			if c.plcErrSuppressed > 0 {
				log.Printf("opus PLC decode error (non-fatal): %v (suppressed %d similar errors)", err, c.plcErrSuppressed)
				c.plcErrSuppressed = 0
			} else {
				log.Printf("opus PLC decode error (non-fatal): %v", err)
			}
			c.lastPLCErrLog = now
		} else {
			c.plcErrSuppressed++
		}
		return nil
	}

	byteLen := FrameBytes
	return unsafe.Slice((*byte)(unsafe.Pointer(&c.decBuf[0])), byteLen)
}
