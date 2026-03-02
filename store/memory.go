package store

import (
	"fmt"
	"sync"

	"github.com/betim/goqueue/queue"
)

// MemoryStore keeps jobs in a map protected by a read-write mutex.
// This is the same storage logic that was originally inside Manager.
type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]*queue.Job
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[string]*queue.Job),
	}
}

func (s *MemoryStore) Save(job *queue.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	return nil
}

func (s *MemoryStore) Get(id string) (*queue.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job %s not found", id)
	}
	return job, nil
}

func (s *MemoryStore) List(status queue.Status) ([]*queue.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*queue.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		if status == "" || job.Status == status {
			result = append(result, job)
		}
	}
	return result, nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[id]; !ok {
		return fmt.Errorf("job %s not found", id)
	}
	delete(s.jobs, id)
	return nil
}

func (s *MemoryStore) Stats() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]int{
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
		"total":     len(s.jobs),
	}
	for _, job := range s.jobs {
		stats[string(job.Status)]++
	}
	return stats, nil
}
