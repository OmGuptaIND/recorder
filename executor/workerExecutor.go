package executor

import (
	"context"
	"log"
	"sync"
	"time"
)

type Job struct {
	Id        string
	Ctx       context.Context
	JobFunc   func() error
	OnError   func(error)
	OnSuccess func()
}

type WorkerExecutorOptions struct {
	MaxRetries   int
	WorkerCount  int
	RetryBackoff time.Duration
}

type WorkerExecutor struct {
	ctx  context.Context
	jobs chan Job
	wg   *sync.WaitGroup
	opts *WorkerExecutorOptions
}

func NewWorkerExecutor(ctx context.Context, opts *WorkerExecutorOptions) *WorkerExecutor {
	return &WorkerExecutor{
		ctx:  ctx,
		jobs: make(chan Job),
		wg:   &sync.WaitGroup{},
		opts: opts,
	}
}

// Enqueue adds a job to the worker queue.
func (w *WorkerExecutor) Enqueue(job Job) {
	w.jobs <- job
}

func (w *WorkerExecutor) Start() {
	for i := 0; i < w.opts.WorkerCount; i++ {
		w.wg.Add(1)

		go func() {
			defer w.wg.Done()
			w.spinWorker()
		}()
	}
}

// Wait for all workers to finish.
func (w *WorkerExecutor) Wait() {
	w.wg.Wait()
}

// Stop the worker queue.
func (w *WorkerExecutor) Stop() {
	if w.jobs == nil {
		return
	}

	close(w.jobs)
	w.jobs = nil
}

// Wroker spins up a worker that processes jobs from the queue.
func (w *WorkerExecutor) spinWorker() {
	for job := range w.jobs {
		log.Println("New Job", job.Id)
		w.processJob(job)
	}
}

// Process the job, retrying if necessary, and calling the appropriate callbacks.
func (w *WorkerExecutor) processJob(job Job) {
	retryBackOff := w.opts.RetryBackoff

	for i := 0; i <= w.opts.MaxRetries; i++ {
		log.Println("Processing job", job.Id)
		err := job.JobFunc()

		if err == nil {
			log.Println("Job", job.Id, "completed successfully")
			if job.Ctx.Err() == nil {
				job.OnSuccess()
			}
			return
		}

		if i == w.opts.MaxRetries {
			if job.Ctx.Err() == nil {
				job.OnError(err)
			}

			return
		}

		if retryBackOff != 0 {
			select {
			case <-time.After(retryBackOff):
				log.Println("Retrying job", job.Id, "after", retryBackOff)

				retryBackOff *= 2
				continue
			case <-job.Ctx.Done():
				err := job.Ctx.Err()

				if err != nil {
					log.Println("Job Context is Done", job.Id, err)
					job.OnError(err)
				}

				return
			}
		}
	}
}
