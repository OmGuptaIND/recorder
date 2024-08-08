package recorder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/OmGuptaIND/display"
	"github.com/google/uuid"
)

type NewRecorderOptions struct {
	FfmpegLogs bool
	StreamUrl  string
	*display.Display
}

type Recorder struct {
	ID  string
	Ctx context.Context

	recordMutex *sync.Mutex
	recordCmd   *exec.Cmd

	streamMutex *sync.Mutex
	streamCmd   *exec.Cmd

	wg        *sync.WaitGroup
	CloseHook func() error

	NewRecorderOptions
}

func NewRecorder(opts NewRecorderOptions) *Recorder {
	return &Recorder{
		ID:                 uuid.New().String(),
		Ctx:                context.Background(),
		wg:                 &sync.WaitGroup{},
		recordMutex:        &sync.Mutex{},
		streamMutex:        &sync.Mutex{},
		NewRecorderOptions: opts,
	}
}

// RecordingPath returns the path where the recording will be saved.
func (r *Recorder) RecordingPath() string {
	return fmt.Sprintf("./out/%s.mp4", r.ID)
}

func (r *Recorder) StartRecording() error {
	r.recordMutex.Lock()
	defer r.recordMutex.Unlock()

	if r.recordCmd != nil {
		return fmt.Errorf("recording already in progress")
	}

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
		"-c:a", "aac",
		"-b:a", "128k",
		"-async", "1",
		"-y",
		r.RecordingPath())

	if r.FfmpegLogs {
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

	return nil
}

// StopRecording sends an interrupt signal to the recording process and waits for it to finish.
func (r *Recorder) StopRecording() error {
	if r.recordCmd == nil {
		log.Println("Recording process is not running")
		return nil
	}

	r.recordMutex.Lock()
	defer r.recordMutex.Unlock()

	log.Println("Stopping Recorder process...")
	if r.recordCmd == nil || r.recordCmd.Process == nil {
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

	if err := r.recordCmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

	log.Println("Recorder process stopped successfully")
	return nil
}

// StartStream starts a new FFmpeg process to stream the display and audio.
func (r *Recorder) StartStream() error {
	r.streamMutex.Lock()
	defer r.streamMutex.Unlock()

	if r.streamCmd != nil {
		log.Println("Stream already in progress")
		return errors.New("stream already in progress")
	}

	log.Println("Starting Stream process...")

	cmd := exec.Command("ffmpeg",
		"-loglevel", "trace",
		"-f", "x11grab",
		"-video_size", "1280x720",
		"-i", r.GetDisplayId(),
		"-f", "pulse",
		"-i", r.GetPulseMonitorId(),
		"-c:v", "libx264", "-preset", "veryslow", "-maxrate", "4500k", "-bufsize", "9000k",
		"-g", "60", "-keyint_min", "60",
		"-c:a", "aac", "-b:a", "160k", "-ar", "44100",
		"-flvflags", "no_duration_filesize",
		"-fflags", "nobuffer",
		"-f", "flv",
		"-rtmp_live", "live",
		"-rtmp_buffer", "3000",
		r.StreamUrl,
	)

	if r.FfmpegLogs {
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
		log.Printf("Failed to start FFmpeg: %v", err)
		return err
	}

	r.streamCmd = cmd

	log.Println("Stream process started successfully")

	return nil
}

// StopStream sends an interrupt signal to the stream process and waits for it to finish.
func (r *Recorder) StopStream() error {
	if r.streamCmd == nil {
		log.Println("Stream process is not running")
		return nil
	}

	r.streamMutex.Lock()
	defer r.streamMutex.Unlock()
	log.Println("Stopping Stream process...")

	if r.streamCmd == nil || r.streamCmd.Process == nil {
		log.Println("FFmpeg process is not running")
		return nil
	}

	if err := r.streamCmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

	log.Println("Stream process stopped successfully")
	return nil
}

// Close stops the recording and stream processes and waits for them to finish.
func (r *Recorder) Close() error {
	if err := r.StopRecording(); err != nil {
		return err
	}

	if err := r.StopStream(); err != nil {
		return err
	}

	r.wg.Wait()

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
