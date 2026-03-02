package queue

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Store is the interface the Manager uses for persistence.
// Defined here (in the consumer package) following Go convention:
// "Accept interfaces, return structs."
type Store interface {
	Save(job *Job) error
	Get(id string) (*Job, error)
	List(status Status) ([]*Job, error)
	Delete(id string) error
	Stats() (map[string]int, error)
}

// Manager coordinates job creation, dispatch, and persistence.
// It owns the job channel but delegates storage to a Store.
type Manager struct {
	store   Store
	JobChan chan *Job
}

// NewManager creates a Manager backed by the given store.
func NewManager(bufferSize int, store Store) *Manager {
	return &Manager{
		store:   store,
		JobChan: make(chan *Job, bufferSize),
	}
}

// AddJob creates a new job, persists it, and sends it to workers.
func (m *Manager) AddJob(jobType string, payload []byte) (*Job, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate job ID: %w", err)
	}

	job := &Job{
		ID:        id,
		Type:      jobType,
		Payload:   payload,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}

	if err := m.store.Save(job); err != nil {
		return nil, fmt.Errorf("save job: %w", err)
	}

	m.JobChan <- job

	return job, nil
}

// GetJob returns a single job by ID.
func (m *Manager) GetJob(id string) (*Job, error) {
	return m.store.Get(id)
}

// ListJobs returns all jobs, optionally filtered by status.
func (m *Manager) ListJobs(status Status) []*Job {
	jobs, _ := m.store.List(status)
	if jobs == nil {
		return []*Job{}
	}
	return jobs
}

// DeleteJob cancels a pending job by removing it from the store.
func (m *Manager) DeleteJob(id string) error {
	job, err := m.store.Get(id)
	if err != nil {
		return err
	}
	if job.Status != StatusPending {
		return fmt.Errorf("can only cancel pending jobs, current status: %s", job.Status)
	}
	return m.store.Delete(id)
}

// RetryJob requeues a failed job by resetting its status and re-dispatching.
func (m *Manager) RetryJob(id string) (*Job, error) {
	job, err := m.store.Get(id)
	if err != nil {
		return nil, err
	}
	if job.Status != StatusFailed {
		return nil, fmt.Errorf("can only retry failed jobs, current status: %s", job.Status)
	}

	job.Status = StatusPending
	job.Error = ""
	job.Result = ""
	job.StartedAt = time.Time{}
	job.EndedAt = time.Time{}

	if err := m.store.Save(job); err != nil {
		return nil, fmt.Errorf("save job: %w", err)
	}

	m.JobChan <- job

	return job, nil
}

// Stats returns counts of jobs by status.
func (m *Manager) Stats() map[string]int {
	stats, _ := m.store.Stats()
	if stats == nil {
		return map[string]int{}
	}
	return stats
}

// generateID creates a random hex string (simplified UUID).
func generateID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
