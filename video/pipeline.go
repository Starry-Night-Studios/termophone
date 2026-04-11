package video

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func StartScreenShare(ctx context.Context, h host.Host, peerID peer.ID, quality string) error {
	if _, ok := Qualities[quality]; !ok {
		quality = "medium"
	}
	q := Qualities[quality]

	s, err := h.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return err
	}

	cmd, stdout, err := buildFFmpegForShare(ctx, q)
	if err != nil {
		s.Reset()
		return err
	}

	go func() {
		defer s.Close()
		defer killProcess(cmd)
		if err := videoWriter(stdout, s); err != nil && err != io.EOF {
			log.Printf("Screen share stopped: %v", err)
		}
		_ = cmd.Wait()
	}()

	return nil
}

func ReceiveScreenShare(ctx context.Context, s network.Stream) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.CommandContext(ctx, "mpv",
			"--no-cache",
			"--untimed",
			"--profile=low-latency",
			"--demuxer-lavf-buffersize=32768",
			"fd://0",
		)
	} else {
		if _, err := exec.LookPath("gst-launch-1.0"); err != nil {
			s.Reset()
			return errors.New("gst-launch-1.0 not found in PATH. Install GStreamer with the Complete option selected. On Windows, also add C:\\gstreamer\\1.0\\msvc_x86_64\\bin to your system PATH")
		}
		cmd = exec.CommandContext(ctx, "gst-launch-1.0",
			"fdsrc", "fd=0", "!",
			"h264parse", "!",
			"avdec_h264", "!",
			"autovideosink", "sync=false",
		)
		cmd.Stderr = os.Stderr
	}

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
		if err := videoReader(s, stdin); err != nil && err != io.EOF {
			log.Printf("Screen share viewing stopped: %v", err)
		}
		_ = stdin.Close()
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}()

	return nil
}

func killProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
