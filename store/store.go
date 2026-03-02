// Package store defines the storage interface for jobs and provides implementations.
package store

import "github.com/betim/goqueue/queue"

// Store is the interface that any job storage backend must satisfy.
// The Manager delegates all persistence to a Store, so swapping
// backends (memory, SQLite, Postgres) only requires a new implementation.
type Store interface {
	// Save persists a job. It creates a new record or updates an existing one.
	Save(job *queue.Job) error

	// Get returns a single job by ID.
	Get(id string) (*queue.Job, error)

	// List returns all jobs, optionally filtered by status (empty string = all).
	List(status queue.Status) ([]*queue.Job, error)

	// Delete removes a job by ID.
	Delete(id string) error

	// Stats returns counts of jobs grouped by status.
	Stats() (map[string]int, error)
}
