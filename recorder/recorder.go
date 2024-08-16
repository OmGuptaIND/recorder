package recorder

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/OmGuptaIND/display"
)

type NewRecorderOptions struct {
	ID             string
	ShowFfmpegLogs bool
	Ctx            context.Context
	Wg             *sync.WaitGroup
	*display.Display
}

type Recorder struct {
	mtx       *sync.Mutex
	recordCmd *exec.Cmd

	CloseHook func() error

	done   chan bool
	Closed bool

	*NewRecorderOptions
}

func NewRecorder(opts NewRecorderOptions) *Recorder {
	return &Recorder{
		mtx:                &sync.Mutex{},
		done:               make(chan bool, 1),
		NewRecorderOptions: &opts,
	}
}

// Done returns a channel that will be closed when the recording is done.
func (r *Recorder) Done() <-chan bool {
	return r.done
}

// RecordingPath returns the path where the recording will be saved.
func (r *Recorder) RecordingPath() string {
	return fmt.Sprintf("./out/%s.mp4", r.ID)
}

func (r *Recorder) StartRecording() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.recordCmd != nil {
		return fmt.Errorf("recording already in progress")
	}

	log.Println("Starting Recorder process...")
	videoSize := fmt.Sprintf("%dx%d", r.GetWidth(), r.GetHeight())

	cmd := exec.Command("ffmpeg",
		"-nostdin",
		"-loglevel", "trace",
		"-video_size", videoSize,
		"-f", "x11grab",
		"-i", r.GetDisplayId(),
		"-f", "pulse",
		"-i", r.GetPulseMonitorId(),
		"-c:v", "libx264",
		"-vf", "scale=1280:720",
		"-preset", "veryslow",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-async", "1",
		"-y",
		r.RecordingPath())

	if r.ShowFfmpegLogs {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Failed to create stderr pipe: %v", err)
			return err
		}

		copyOutput := func(writer io.Writer, reader io.Reader, name string) {
			_, err := io.Copy(writer, reader)
			if err != nil && err != io.EOF {
				log.Printf("Error copying %s: %v", name, err)
			}
		}
		go copyOutput(os.Stderr, stderr, "stderr")
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %v", err)
	}

	r.recordCmd = cmd

	log.Println("Recorder process started successfully")

	r.Wg.Add(1)
	go r.HandleContextCancel()

	return nil
}

// Close sends an interrupt signal to the recording process and waits for it to finish.
func (r *Recorder) Close() error {
	if r.recordCmd == nil {
		log.Println("Recording process is not running")
		return nil
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	log.Println("Stopping Recorder process...")

	if r.recordCmd == nil || r.recordCmd.Process == nil {
		log.Println("Recording FFmpeg process is not running")
		return nil
	}

	defer func() {
		if r.CloseHook != nil {
			log.Println("Cleaning Up Recorder CloseHook...")
			if err := r.CloseHook(); err != nil {
				log.Printf("CloseHook failed: %v", err)
			}
		}
	}()

	if err := r.recordCmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

	done := make(chan error, 1)
	timeout := time.After(20 * time.Second)

	r.Wg.Add(1)
	go func() {
		defer r.Wg.Done()
		done <- r.recordCmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() != 255 {
					log.Printf("FFmpeg process exited with status: %d", exitErr.ExitCode())
				}
			}
		}
	case <-timeout:
		log.Println("Recording process did not stop in time, killing it")
		if err := r.recordCmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill Recording process: %v", err)
		}
	}

	log.Println("Recording process stopped")

	r.Closed = true
	r.recordCmd = nil
	close(r.done)

	return nil
}

func (r *Recorder) VerifyOutputFile() error {
	info, err := os.Stat(r.RecordingPath())
	if err != nil {
		return fmt.Errorf("failed to stat output file: %v", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("output file is empty")
	}
	log.Printf("Output file size: %d bytes", info.Size())

	cmd := exec.Command("ffprobe", "-v", "error", r.RecordingPath())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffprobe check failed: %v, output: %s", err, output)
	}

	return nil
}

// HandleContextCancel handles the context cancel signal.
func (r *Recorder) HandleContextCancel() {
	defer r.Wg.Done()
	<-r.Ctx.Done()
	log.Println("Context Done, Stopping Recorder")
	r.Close()
}
