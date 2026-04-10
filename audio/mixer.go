package audio

// MixAudio receives a slice of decoded int16 PCM streams (one per active peer)
// and sums them into a single output buffer using float32 math and cubic soft-clipping.
func MixAudio(peerStreams [][]int16, frameSize int) []int16 {
	output := make([]int16, frameSize)

	if len(peerStreams) == 0 {
		return output // Return silence (zeros) if no streams
	}

	for i := 0; i < frameSize; i++ {
		var sum float32 = 0.0

		// 1. Normalize & Sum
		// Summing the decibel equivalent without integer wrapping concerns
		for _, stream := range peerStreams {
			if i < len(stream) {
				// Normalize int16 (-32768 to 32767) to float32 (-1.0 to 1.0)
				sum += float32(stream[i]) / 32768.0
			}
		}

		// 2. Soft-Clip (Cubic)
		// y = 1.5x - 0.5x^3
		var clipped float32
		if sum <= -1.0 {
			clipped = -1.0
		} else if sum >= 1.0 {
			clipped = 1.0
		} else {
			clipped = 1.5*sum - 0.5*(sum*sum*sum)
		}

		// 3. Denormalize
		// Cast back to the safe bounds of 16-bit integer PCM array
		output[i] = int16(clipped * 32767.0)
	}

	return output
}
