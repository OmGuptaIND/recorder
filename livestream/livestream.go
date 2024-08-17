package livestream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/OmGuptaIND/display"
	"github.com/google/uuid"
)

type NewLivestreamOptions struct {
	ShowFfmpegLogs bool
	StreamUrl      string
	Ctx            context.Context
	Wg             *sync.WaitGroup

	*display.Display
}

type Livestream struct {
	ID string

	mtx       *sync.Mutex
	streamCmd *exec.Cmd
	closeHook func() error

	done   chan bool
	Closed bool

	*NewLivestreamOptions
}

// NewLivestream initializes a new Livestream with the specified options.
func NewLivestream(opts NewLivestreamOptions) *Livestream {
	return &Livestream{
		ID:                   uuid.New().String(),
		mtx:                  &sync.Mutex{},
		done:                 make(chan bool, 1),
		NewLivestreamOptions: &opts,
	}
}

// Done returns a channel that will be closed when the stream is done.
func (l *Livestream) Done() <-chan bool {
	return l.done
}

// startStream starts the stream.
func (l *Livestream) StartStream() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.streamCmd != nil {
		return errors.New("stream already in progress")
	}

	log.Println("Staring Live Streaming")

	l.Wg.Add(1)
	go l.HandleContextCancel()

	cmd := exec.Command("ffmpeg",
		"-nostdin",
		"-loglevel", "trace",
		"-f", "x11grab",
		"-video_size", "1280x720",
		"-i", l.GetDisplayId(),
		"-f", "pulse",
		"-i", l.GetPulseMonitorId(),
		"-c:v", "libx264", "-preset", "veryslow", "-maxrate", "4500k", "-bufsize", "9000k",
		"-g", "60", "-keyint_min", "60",
		"-c:a", "aac", "-b:a", "160k", "-ar", "44100",
		"-flvflags", "no_duration_filesize",
		"-fflags", "nobuffer",
		"-f", "flv",
		"-rtmp_live", "live",
		"-rtmp_buffer", "3000",
		l.StreamUrl,
	)

	if l.ShowFfmpegLogs {
		stderr, err := cmd.StderrPipe()

		if err != nil {
			log.Printf("Failed to get ffmpeg logs: %v", err)
		} else {
			copyOutput := func(writer io.Writer, reader io.Reader, name string) {
				_, err := io.Copy(writer, reader)
				if err != nil && err != io.EOF {
					log.Printf("Error copying %s: %v", name, err)
				}
			}
			go copyOutput(os.Stderr, stderr, "stderr")
		}
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start FFmpeg: %v", err)
		return err
	}

	l.streamCmd = cmd

	log.Println("Stream process started successfully")

	return nil
}

// StopStream stops the stream.
func (l *Livestream) Close() error {
	if l.streamCmd == nil {
		log.Println("Stream is not running")
		return nil
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	log.Println("Stopping Stream process...")

	defer func() {
		if l.closeHook != nil {
			if err := l.closeHook(); err != nil {
				log.Printf("Error in Livestream closeHook: %v", err)
			}
		}
	}()

	if l.streamCmd == nil || l.streamCmd.Process == nil {
		log.Println("Streaming FFmpeg process is not running")
		return nil
	}

	if err := l.streamCmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

	done := make(chan error, 1)
	timeout := time.After(10 * time.Second)

	l.Wg.Add(1)
	go func() {
		defer l.Wg.Done()
		done <- l.streamCmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() != 255 || exitErr.ExitCode() != -1 {
					log.Printf("FFmpeg process exited with status: %d", exitErr.ExitCode())
				}
			}
		}
	case <-timeout:
		log.Println("Stream process did not stop in time, killing it")
		if err := l.streamCmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill stream process: %v", err)
		}
	}

	log.Println("Live Stream process stopped")

	l.streamCmd = nil
	l.Closed = true
	close(l.done)

	return nil
}

// HandleContextCancel handles the context cancel signal.
func (l *Livestream) HandleContextCancel() {
	defer l.Wg.Done()
	<-l.Ctx.Done()
	log.Println("Context Done, Stopping Stream")
	l.Close()
}
