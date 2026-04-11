package video

const ProtocolID = "/termophone/screen/1.0.0"

var h264StartCode = []byte{0, 0, 0, 1}

// Temporary debugging toggle for ffmpeg stderr.
const debugFFmpeg = false

type Quality struct {
	Name    string
	Scale   string
	Bitrate string
}

var Qualities = map[string]Quality{
	"low":    {"Low", "scale=-2:480", "500k"},
	"medium": {"Medium", "scale=-2:720", "1500k"},
	"high":   {"High", "scale=-2:1080", "4000k"},
}
