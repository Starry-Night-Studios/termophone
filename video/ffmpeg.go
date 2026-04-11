package video

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func buildFFmpegForShare(ctx context.Context, q Quality) (*exec.Cmd, io.ReadCloser, error) {
	if runtime.GOOS == "darwin" {
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-f", "avfoundation",
			"-capture_cursor", "1",
			"-capture_mouse_clicks", "1",
			"-pixel_format", "uyvy422",
			"-framerate", "30",
			"-video_size", "1920x1200",
			"-i", "1:none",
			"-r", "30",
			"-vf", q.Scale,
			"-c:v", "h264_videotoolbox",
			"-realtime", "1",
			"-allow_sw", "0",
			"-intra-refresh", "1",
			"-g", "30",
			"-maxrate", q.Bitrate,
			"-bufsize", q.Bitrate,
			"-f", "h264",
			"pipe:1",
		)
		out, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, err
		}
		if debugFFmpeg {
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		return cmd, out, nil
	}

	if runtime.GOOS == "windows" {
		baseArgs := []string{
			"-f", "gdigrab", "-i", "desktop",
			"-r", "30",
			"-vf", q.Scale,
		}
		encoderProfiles := []struct {
			name string
			args []string
		}{
			{
				name: "h264_nvenc",
				args: []string{
					"-c:v", "h264_nvenc",
					"-preset", "p2",
					"-rc", "cbr_ld_hq",
					"-bf", "0",
					"-delay", "0",
					"-zerolatency", "1",
					"-g", "30",
					"-maxrate", q.Bitrate,
					"-bufsize", q.Bitrate,
					"-f", "h264",
					"pipe:1",
				},
			},
			{
				name: "h264_qsv",
				args: []string{
					"-c:v", "h264_qsv",
					"-preset", "veryfast",
					"-bf", "0",
					"-g", "30",
					"-maxrate", q.Bitrate,
					"-bufsize", q.Bitrate,
					"-f", "h264",
					"pipe:1",
				},
			},
			{
				name: "h264_amf",
				args: []string{
					"-c:v", "h264_amf",
					"-usage", "ultralowlatency",
					"-bf", "0",
					"-g", "30",
					"-maxrate", q.Bitrate,
					"-bufsize", q.Bitrate,
					"-f", "h264",
					"pipe:1",
				},
			},
			{
				name: "libx264",
				args: []string{
					"-c:v", "libx264",
					"-preset", "ultrafast",
					"-tune", "zerolatency",
					"-intra-refresh", "1",
					"-g", "30",
					"-maxrate", q.Bitrate,
					"-bufsize", q.Bitrate,
					"-f", "h264",
					"pipe:1",
				},
			},
		}

		var lastErr error
		for _, profile := range encoderProfiles {
			args := make([]string, 0, len(baseArgs)+len(profile.args))
			args = append(args, baseArgs...)
			args = append(args, profile.args...)

			attempt := exec.CommandContext(ctx, "ffmpeg", args...)
			if debugFFmpeg {
				attempt.Stderr = os.Stderr
			}

			out, err := attempt.StdoutPipe()
			if err != nil {
				lastErr = err
				continue
			}

			if err := attempt.Start(); err != nil {
				lastErr = err
				log.Printf("video encoder %s unavailable, trying fallback: %v", profile.name, err)
				continue
			}

			log.Printf("video encoder selected: %s", profile.name)
			return attempt, out, nil
		}

		if lastErr == nil {
			lastErr = errors.New("no ffmpeg encoder profile could start")
		}
		return nil, nil, lastErr
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "x11grab", "-i", ":0.0",
		"-r", "30",
		"-vf", q.Scale,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-intra-refresh", "1",
		"-g", "30",
		"-maxrate", q.Bitrate,
		"-bufsize", q.Bitrate,
		"-f", "h264",
		"pipe:1",
	)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if debugFFmpeg {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, out, nil
}
