package queue

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// Manager holds all jobs in memory and provides a channel for workers to consume from.
type Manager struct {
	jobs    map[string]*Job
	mu      sync.RWMutex
	JobChan chan *Job
}

// NewManager creates a Manager with the given queue buffer size.
func NewManager(bufferSize int) *Manager {
	return &Manager{
		jobs:    make(map[string]*Job),
		JobChan: make(chan *Job, bufferSize),
	}
}

// AddJob creates a new job and sends it to the job channel for workers to pick up.
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

	m.mu.Lock()
	m.jobs[id] = job
	m.mu.Unlock()

	m.JobChan <- job

	return job, nil
}

// GetJob returns a single job by ID.
func (m *Manager) GetJob(id string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job %s not found", id)
	}
	return job, nil
}

// ListJobs returns all jobs, optionally filtered by status.
func (m *Manager) ListJobs(status Status) []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Job, 0)
	for _, job := range m.jobs {
		if status == "" || job.Status == status {
			result = append(result, job)
		}
	}
	return result
}

// DeleteJob cancels a pending job by removing it from the store.
func (m *Manager) DeleteJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return fmt.Errorf("job %s not found", id)
	}
	if job.Status != StatusPending {
		return fmt.Errorf("can only cancel pending jobs, current status: %s", job.Status)
	}

	delete(m.jobs, id)
	return nil
}

// RetryJob requeues a failed job by resetting its status and sending it to the channel.
func (m *Manager) RetryJob(id string) (*Job, error) {
	m.mu.Lock()

	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("job %s not found", id)
	}
	if job.Status != StatusFailed {
		m.mu.Unlock()
		return nil, fmt.Errorf("can only retry failed jobs, current status: %s", job.Status)
	}

	job.Status = StatusPending
	job.Error = ""
	job.Result = ""
	job.StartedAt = time.Time{}
	job.EndedAt = time.Time{}

	m.mu.Unlock()

	m.JobChan <- job

	return job, nil
}

// Stats returns counts of jobs by status.
func (m *Manager) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
		"total":     len(m.jobs),
	}
	for _, job := range m.jobs {
		stats[string(job.Status)]++
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
