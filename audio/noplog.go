package audio

import (
	"io"
	"log"
	"strings"
)

type mdnsFilter struct {
	target io.Writer
}

func (m *mdnsFilter) Write(p []byte) (n int, err error) {
	if strings.Contains(string(p), "[WARN] mdns: Failed to set multicast interface") {
		// absorb zeroconf noise
		return len(p), nil
	}
	return m.target.Write(p)
}

func InitFilterLog(w io.Writer) {
	log.SetOutput(&mdnsFilter{target: w})
}
