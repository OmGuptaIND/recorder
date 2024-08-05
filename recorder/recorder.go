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
	"github.com/google/uuid"
)

type NewRecorderOptions struct {
	*display.Display
}

type Recorder struct {
	ID          string
	Ctx         context.Context
	ffmpeg      *exec.Cmd
	ffmpegStdin io.WriteCloser
	wg          *sync.WaitGroup
	CloseHook   func() error

	NewRecorderOptions
}

func NewRecorder(opts NewRecorderOptions) *Recorder {
	return &Recorder{
		ID:                 uuid.New().String(),
		Ctx:                context.Background(),
		wg:                 &sync.WaitGroup{},
		NewRecorderOptions: opts,
	}
}

// RecordingPath returns the path where the recording will be saved.
func (r *Recorder) RecordingPath() string {
	return fmt.Sprintf("./out/%s.mp4", r.ID)
}

func (r *Recorder) StartRecording() error {
	log.Println("Starting Recorder process...")
	videoSize := fmt.Sprintf("%dx%d", r.GetWidth(), r.GetHeight())

	cmd := exec.Command("ffmpeg",
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
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
		"-async", "1",
		"-y",
		r.RecordingPath())

	ioCloser, err := cmd.StdinPipe()

	if err != nil {
		return fmt.Errorf("failed to open stdin pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %v", err)
	}

	copyOutput := func(writer io.Writer, reader io.Reader, name string) {
		_, err := io.Copy(writer, reader)
		if err != nil && err != io.EOF {
			log.Printf("Error copying %s: %v", name, err)
		}
	}

	// Start goroutines to copy stdout and stderr
	go copyOutput(os.Stderr, stderr, "stderr")

	r.ffmpeg = cmd
	r.ffmpegStdin = ioCloser

	return nil
}

func (r *Recorder) StopRecording() error {
	log.Println("Stopping Recorder process...")
	if r.ffmpeg == nil || r.ffmpeg.Process == nil {
		log.Println("FFmpeg process is not running")
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

	ctx, cancel := context.WithTimeout(r.Ctx, 10*time.Second)
	defer cancel()

	v, err := r.ffmpegStdin.Write([]byte("q"))
	defer r.ffmpegStdin.Close()

	if err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

	log.Println("Interrupt signal sent to FFmpeg", v)

	done := make(chan error, 1)
	go func() {
		err := r.ffmpeg.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("FFmpeg exited with error: %v", err)
			return err
		}
	case <-ctx.Done():
		log.Println("FFmpeg didn't exit in time, force killing...")
		err := r.ffmpeg.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to kill FFmpeg process: %v", err)
		}
	}

	if err := r.verifyOutputFile(); err != nil {
		return fmt.Errorf("output file verification failed: %v", err)
	}

	log.Println("Recorder process stopped successfully")
	return nil
}

func (r *Recorder) verifyOutputFile() error {
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
