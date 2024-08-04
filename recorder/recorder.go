package recorder

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type NewRecorderOptions struct {
	Width   int
	Height  int
	Depth   int
	Display string `yaml:"-"`
}

type Recorder struct {
	Ctx    context.Context
	ffmpeg *exec.Cmd
	opts   NewRecorderOptions
}

func NewRecorder(opts NewRecorderOptions) *Recorder {
	return &Recorder{
		opts: opts,
		Ctx:  context.Background(),
	}
}

func (r *Recorder) StartRecording() error {
	log.Println("Starting Recorder process...")
	videoSize := fmt.Sprintf("%dx%d", r.opts.Width, r.opts.Height)

	cmd := exec.Command("ffmpeg",
		"-loglevel", "debug",
		"-video_size", videoSize,
		"-framerate", "25",
		"-f", "x11grab",
		"-i", fmt.Sprintf("%s.0", r.opts.Display),
		"-c:v", "libx264", //
		"-vf", "scale=1280:720",
		"-preset", "ultrafast",
		"-crf", "23",
		"-pix_fmt", "yuv420p",
		"output.mp4")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %v", err)
	}

	r.ffmpeg = cmd

	return nil
}

func (r *Recorder) StopRecording() error {
	log.Println("Stopping Recorder process...")
	if r.ffmpeg == nil || r.ffmpeg.Process == nil {
		log.Println("FFmpeg process is not running")
		return nil
	}

	ctx, cancel := context.WithTimeout(r.Ctx, 10*time.Second)
	defer cancel()

	if err := r.ffmpeg.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

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
		if err := r.ffmpeg.Process.Kill(); err != nil {
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
	info, err := os.Stat("output.mp4")
	if err != nil {
		return fmt.Errorf("failed to stat output file: %v", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("output file is empty")
	}
	log.Printf("Output file size: %d bytes", info.Size())

	cmd := exec.Command("ffprobe", "-v", "error", "output.mp4")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffprobe check failed: %v, output: %s", err, output)
	}

	return nil
}
