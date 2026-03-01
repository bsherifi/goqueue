package queue

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// WorkerPool runs N goroutines that consume jobs from a channel.
type WorkerPool struct {
	numWorkers int
	jobChan    chan *Job
	wg         sync.WaitGroup
}

// NewWorkerPool creates a pool with the given number of workers and job channel.
func NewWorkerPool(numWorkers int, jobChan chan *Job) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobChan:    jobChan,
	}
}

// Start launches all workers. They stop when the context is cancelled.
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := range wp.numWorkers {
		wp.wg.Add(1)
		go wp.run(ctx, i)
	}
	fmt.Printf("Started %d workers\n", wp.numWorkers)
}

// Stop waits for all workers to finish their current job and exit.
func (wp *WorkerPool) Stop() {
	wp.wg.Wait()
	fmt.Println("All workers stopped")
}

func (wp *WorkerPool) run(ctx context.Context, id int) {
	defer wp.wg.Done()
	fmt.Printf("Worker %d: started\n", id)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d: shutting down\n", id)
			return
		case job, ok := <-wp.jobChan:
			if !ok {
				fmt.Printf("Worker %d: channel closed\n", id)
				return
			}
			wp.process(id, job)
		}
	}
}

func (wp *WorkerPool) process(workerID int, job *Job) {
	fmt.Printf("Worker %d: processing job %s (type: %s)\n", workerID, job.ID, job.Type)

	job.Status = StatusRunning
	job.StartedAt = time.Now()

	// Simulate work: sleep 1-3 seconds
	duration := time.Duration(1+rand.Intn(3)) * time.Second
	time.Sleep(duration)

	// Simulate occasional failures (20% chance)
	if rand.Intn(5) == 0 {
		job.Status = StatusFailed
		job.Error = "simulated random failure"
		job.EndedAt = time.Now()
		fmt.Printf("Worker %d: job %s failed (%s)\n", workerID, job.ID, duration)
		return
	}

	job.Status = StatusCompleted
	job.Result = fmt.Sprintf("processed by worker %d in %s", workerID, duration)
	job.EndedAt = time.Now()
	fmt.Printf("Worker %d: job %s completed (%s)\n", workerID, job.ID, duration)
}
