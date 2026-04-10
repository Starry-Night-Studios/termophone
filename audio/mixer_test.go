package audio

import (
	"testing"
)

func TestMixAudio_ZeroStreams(t *testing.T) {
	frameSize := 10
	out := MixAudio(nil, frameSize)

	if len(out) != frameSize {
		t.Fatalf("expected output length %d, got %d", frameSize, len(out))
	}
	for i, v := range out {
		if v != 0 {
			t.Errorf("expected silence (0) at %d, got %d", i, v)
		}
	}
}

func TestMixAudio_Clipping(t *testing.T) {
	// Let's create an extreme scenario where 3 people are screaming
	// which would normally overflow int16 (max 32767)

	p1 := []int16{30000, 30000}
	p2 := []int16{30000, 30000}
	p3 := []int16{30000, 30000}

	streams := [][]int16{p1, p2, p3}
	frameSize := 2

	out := MixAudio(streams, frameSize)

	// Since sum is huge (90000 / 32768.0 = ~2.7), it should hit the hard limit of the cubic math (+1.0)
	// which denormalizes to 32767
	for i, v := range out {
		if v != 32767 {
			t.Errorf("expected saturated peak 32767 at %d, got %d", i, v)
		}
	}
}

func TestMixAudio_Summation(t *testing.T) {
	// A normal volume summation where signals just add up
	// and stay within the safe bounds of cubic saturation
	p1 := []int16{10000} // ~0.3
	p2 := []int16{10000} // ~0.3

	streams := [][]int16{p1, p2}
	frameSize := 1

	out := MixAudio(streams, frameSize)

	// Sum = 20000/32768.0 = 0.61035
	// Cubic Equation constraint check
	// y = 1.5 * 0.61035 - 0.5 * (0.61035^3) = ~0.915525 - 0.1136 = ~0.801
	// 0.801 * 32767 = ~26246

	if out[0] < 26000 || out[0] > 26500 {
		t.Errorf("Expected soft-clipped sum around 26246, got %d", out[0])
	}
}
