package recorder

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/OmGuptaIND/config"
	"github.com/OmGuptaIND/display"
	"github.com/OmGuptaIND/pkg"
)

type ChunkInfo struct {
	Filename string  `json:"filename"`
	PTS      float64 `json:"pts_time"`
	Duration float64 `json:"duration"`
}

type ChunkingOptions struct {
	ChunkDuration string
}

type NewRecorderOptions struct {
	ID             string
	ShowFfmpegLogs bool
	Ctx            context.Context
	Wg             *sync.WaitGroup
	Chunking       *ChunkingOptions
	*display.Display
}

type Recorder struct {
	mtx       *sync.Mutex
	recordCmd *exec.Cmd

	CloseHook func() error

	done   chan error
	Closed bool

	*NewRecorderOptions
}

// NewRecorder creates a new Recorder instance.
func NewRecorder(opts NewRecorderOptions) *Recorder {
	recorder := &Recorder{
		mtx:                &sync.Mutex{},
		done:               make(chan error, 1),
		NewRecorderOptions: &opts,
	}

	path := recorder.GetRecordinDirectoryPath()

	log.Println("Recording Directory Path: ", path)

	return recorder
}

// Done returns a channel that will be closed when the recording is done.
func (r *Recorder) Done() <-chan error {
	return r.done
}

// ChunkingDuration returns the duration of each chunk.
func (r *Recorder) ChunkingDuration() string {
	return r.Chunking.ChunkDuration
}

// GetRecordinDirectoryPath returns the path where the recording will be saved.
func (r *Recorder) GetRecordinDirectoryPath() string {
	cwd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Failed to get current working directory, %v", err)
	}

	path := filepath.Join(cwd, config.RECORDING_DIR, r.ID)

	pkg.CreateDirectory(path)

	return path
}

// RecordingPath returns the path where the recording will be saved.
func (r *Recorder) recordingFilePath() string {
	return filepath.Join(r.GetRecordinDirectoryPath(), fmt.Sprintf("%s.mp4", r.ID))
}

// RecordingChunkPath returns the path where the recording chunk will be saved.
func (r *Recorder) recordingChunkPath() string {
	return filepath.Join(r.GetRecordinDirectoryPath(), "chunk_%05d.mp4")
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
		"-preset", "veryslow",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-async", "1",
		"-f", "segment",
		"-segment_start_number", "0",
		"-segment_list", "out.list",
		"-segment_time", r.ChunkingDuration(),
		"-segment_format", "mp4",
		"-reset_timestamps", "1",
		"-y",
		r.recordingChunkPath())

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

	r.Wg.Add(1)
	go func() {
		defer r.Wg.Done()

		if err := cmd.Wait(); err != nil {
			r.done <- err
		}
	}()

	r.recordCmd = cmd

	log.Println("Recorder process started successfully")

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

	r.Closed = true
	r.recordCmd = nil
	close(r.done)

	return nil
}

func (r *Recorder) VerifyOutputFile() error {
	info, err := os.Stat(r.recordingFilePath())
	if err != nil {
		return fmt.Errorf("failed to stat output file: %v", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("output file is empty")
	}
	log.Printf("Output file size: %d bytes", info.Size())

	cmd := exec.Command("ffprobe", "-v", "error", r.recordingFilePath())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffprobe check failed: %v, output: %s", err, output)
	}

	return nil
}

// handleContextCancel handles the context cancel signal.
func (r *Recorder) handleContextCancel() {
	defer r.Wg.Done()
	<-r.Ctx.Done()
	log.Println("Context Done, Stopping Recorder")
	r.Close()
}
