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
	close(w.jobs)
}

// Wroker spins up a worker that processes jobs from the queue.
func (w *WorkerExecutor) spinWorker() {
	for {
		select {
		case job := <-w.jobs:
			select {
			case <-job.Ctx.Done():
				err := job.Ctx.Err()

				if err == context.Canceled {
					log.Println("job context is cancelled", job.Id)
					job.OnError(err)
					return
				}

				if err == context.DeadlineExceeded {
					log.Println("job context deadline exceeded", job.Id)
					job.OnError(err)
					return
				}

				w.processJob(job)

			case <-w.ctx.Done():
				log.Println("worker context is done")
				job.OnError(w.ctx.Err())
			}
		case <-w.ctx.Done():
			log.Println("worker context is done")
			for job := range w.jobs {
				job.OnError(w.ctx.Err())
			}
			return
		}
	}
}

// Process the job, retrying if necessary, and calling the appropriate callbacks.
func (w *WorkerExecutor) processJob(job Job) {
	retryBackOff := w.opts.RetryBackoff

	for i := 0; i <= w.opts.MaxRetries; i++ {
		err := job.JobFunc()

		if err == nil {
			log.Println("Job", job.Id, "completed successfully")
			job.OnSuccess()
			return
		}

		if i == w.opts.MaxRetries {
			job.OnError(err)
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
