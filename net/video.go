package net

import (
	"context"
	"io"
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const VideoProtocolID = "/termophone/screen/1.0.0"

type Quality struct {
	Name    string
	Scale   string
	Bitrate string
}

var Qualities = map[string]Quality{
	"low":    {"Low", "scale=-1:480", "500k"},
	"medium": {"Medium", "scale=-1:720", "1500k"},
	"high":   {"High", "scale=-1:1080", "4000k"},
}

func StartScreenShare(ctx context.Context, h host.Host, peerID peer.ID, quality string) error {
	if _, ok := Qualities[quality]; !ok {
		quality = "medium" // default to medium if invalid
	}
	q := Qualities[quality]

	s, err := h.NewStream(ctx, peerID, VideoProtocolID)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-f", "avfoundation",
			"-capture_cursor", "1",
			"-capture_mouse_clicks", "1",
			"-pixel_format", "uyvy422",
			"-i", "1:none",
			"-r", "30",
			"-g", "30",
			"-vf", q.Scale,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
			"-b:v", q.Bitrate,
			"-f", "mpegts",
			"pipe:1",
		)
	} else if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-f", "gdigrab", "-i", "desktop",
			"-r", "30",
			"-g", "30",
			"-vf", q.Scale,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
			"-b:v", q.Bitrate,
			"-f", "mpegts",
			"pipe:1",
		)
	} else {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-f", "x11grab", "-i", ":0.0",
			"-r", "30",
			"-g", "30",
			"-vf", q.Scale,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
			"-b:v", q.Bitrate,
			"-f", "mpegts",
			"pipe:1",
		)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.Reset()
		return err
	}

	if err := cmd.Start(); err != nil {
		s.Reset()
		return err
	}

	go func() {
		defer s.Close()
		defer cmd.Process.Kill()
		if _, err := io.Copy(s, stdout); err != nil {
			log.Printf("Screen share stopped: %v", err)
		}
		cmd.Wait()
	}()

	return nil
}

func ReceiveScreenShare(ctx context.Context, s network.Stream) error {
	cmd := exec.CommandContext(ctx, "mpv",
		"--no-cache",
		"--untimed",
		"--profile=low-latency",
		"-",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.Reset()
		return err
	}

	if err := cmd.Start(); err != nil {
		s.Reset()
		return err
	}

	go func() {
		defer s.Close()
		if _, err := io.Copy(stdin, s); err != nil {
			log.Printf("Screen share viewing stopped: %v", err)
		}
		// Close stdin to signal EOF to mpv
		stdin.Close()
		// Wait for process to exit gracefully, then force kill if needed
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-done:
			// Process exited cleanly
		case <-time.After(2 * time.Second):
			// Force kill if still running after 2 seconds
			cmd.Process.Kill()
			<-done
		}
	}()

	return nil
}
