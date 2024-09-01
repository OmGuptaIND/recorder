package recorder

import (
	"bufio"
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
	Wg             *sync.WaitGroup
	*display.Display
}

type Recorder struct {
	ctx       context.Context
	mtx       *sync.Mutex
	recordCmd *exec.Cmd
	stdout    *io.ReadCloser

	CloseHook func() error

	done chan error

	*NewRecorderOptions
}

// NewRecorder creates a new Recorder instance.
func NewRecorder(ctx context.Context, opts NewRecorderOptions) (*Recorder, error) {
	recorder := &Recorder{
		ctx:                ctx,
		mtx:                &sync.Mutex{},
		done:               make(chan error, 1),
		NewRecorderOptions: &opts,
	}

	return recorder, nil
}

// GetContext returns the context of the Recorder.
func (r *Recorder) GetContext() context.Context {
	return r.ctx
}

// Done returns a channel that will be closed when the recording is done.
func (r *Recorder) Done() <-chan error {
	return r.done
}

// GetRecorderStdout returns the stdout of the recording process.
func (r *Recorder) GetReader() *bufio.Reader {
	log.Println("Getting Recorder stdout...", r.stdout)

	if r.stdout == nil {
		return nil
	}

	reader := bufio.NewReader(*r.stdout)

	return reader
}

// StartRecording starts the recording process.
func (r *Recorder) StartRecording() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.recordCmd != nil {
		return fmt.Errorf("recording already in progress")
	}

	log.Println("Starting Recorder process...")
	r.Wg.Add(1)
	go r.handleContextCancel()

	cmd := exec.Command("ffmpeg",
		"-nostdin",
		"-loglevel", "info",
		"-thread_queue_size", "512",
		"-video_size", fmt.Sprintf("%dx%d", r.GetWidth(), r.GetHeight()),
		"-f", "x11grab",
		"-i", r.GetDisplayId(),
		"-f", "pulse",
		"-i", r.GetPulseMonitorId(),
		"-c:v", "libx264",
		"-vf", "scale=1280:720",
		"-preset", "ultrafast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-async", "1",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-bufsize", "2M",
		"-flush_packets", "1",
		"-y",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	log.Println("Recording Command:", stdout)

	r.stdout = &stdout

	if err := r.showFfmpegLogs(); err != nil {
		return fmt.Errorf("failed to show ffmpeg logs: %v", err)
	}

	r.recordCmd = cmd

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %v", err)
	}

	r.Wg.Add(1)
	go func() {
		defer r.Wg.Done()

		if err := cmd.Wait(); err != nil {
			r.done <- err
		}
	}()

	log.Println("Recorder process started successfully")

	return nil
}

// Close sends an interrupt signal to the recording process and waits for it to finish.
func (r *Recorder) Close() error {
	if r.recordCmd == nil {
		log.Println("Recording process is not running")
		return nil
	}

	if r.ctx.Err() != nil {
		log.Println("Context is already cancelled, no need to stop Recorder")
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

	timeout := time.After(10 * time.Second)

	select {
	case err := <-r.done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() != 255 || exitErr.ExitCode() != -1 {
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

	r.recordCmd = nil
	r.stdout = nil
	close(r.done)

	return nil
}

// handleContextCancel handles the context cancel signal.
func (r *Recorder) handleContextCancel() {
	defer r.Wg.Done()
	<-r.ctx.Done()
	log.Println("Context Done, Stopping Recorder")
	r.Close()
}

// showFfmpegLogs shows the ffmpeg logs.
func (r *Recorder) showFfmpegLogs() error {
	if !r.ShowFfmpegLogs {
		return nil
	}

	stderr, err := r.recordCmd.StderrPipe()

	if err != nil {
		log.Printf("Failed to create stderr pipe: %v", err)
		return err
	}

	r.Wg.Add(1)
	copyOutput := func(writer io.Writer, reader io.Reader, name string) {
		defer func() {
			log.Println("Closing stderr pipe, Log Copy")
			r.Wg.Done()
		}()

		_, err := io.Copy(writer, reader)
		if err != nil && err != io.EOF {
			log.Printf("Error copying %s: %v", name, err)
		}
	}

	go copyOutput(os.Stderr, stderr, "stderr")

	return nil
}
