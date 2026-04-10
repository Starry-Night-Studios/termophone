package net

import (
	"context"
	"io"
	"log"
	"os/exec"
	"runtime"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const VideoProtocolID = "/termophone/screen/1.0.0"

func StartScreenShare(ctx context.Context, h host.Host, peerID peer.ID) error {
	s, err := h.NewStream(ctx, peerID, VideoProtocolID)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-f", "avfoundation", "-i", "1:none",
			"-r", "30",
			"-vf", "scale=-1:480",
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
			"-g", "30",
			"-f", "h264",
			"pipe:1",
		)
	} else if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-f", "gdigrab", "-i", "desktop",
			"-r", "30",
			"-vf", "scale=-1:480",
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
			"-g", "30",
			"-f", "h264",
			"pipe:1",
		)
	} else {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-f", "x11grab", "-i", ":0.0",
			"-r", "30",
			"-vf", "scale=-1:480",
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
			"-g", "30",
			"-f", "h264",
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
		"--demuxer=h264",
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
		defer cmd.Process.Kill()
		if _, err := io.Copy(stdin, s); err != nil {
			log.Printf("Screen share viewing stopped: %v", err)
		}
		cmd.Wait()
	}()

	return nil
}
