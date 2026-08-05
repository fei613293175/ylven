package jobs

import (
	"bytes"
	"context"
	"errors"
	"sync"
)

type State string
const (
	Queued State = "QUEUED"
	Running State = "RUNNING"
	Succeeded State = "SUCCEEDED"
	Failed State = "FAILED"
	Cancelled State = "CANCELLED"
)

var ErrIdempotencyConflict = errors.New("IDEMPOTENCY_CONFLICT")
var ErrStopped = errors.New("worker is stopped")

type Job struct {
	ID string
	IdempotencyKey string
	Payload []byte
	Attempts int
	MaxAttempts int
	State State
	LastError string
}

type ArchiveEntry struct { Job Job; Reason string }

type Store struct {
	mu sync.Mutex
	jobs map[string]Job
	keys map[string]string
	archive []ArchiveEntry
	stopped bool
}

func NewStore() *Store { return &Store{jobs: map[string]Job{}, keys: map[string]string{}} }

func (s *Store) Enqueue(job Job) (Job, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	if s.stopped { return Job{}, ErrStopped }
	if existing, ok := s.keys[job.IdempotencyKey]; ok && existing != job.ID { return Job{}, ErrIdempotencyConflict }
	if existing, ok := s.jobs[job.ID]; ok {
		if existing.IdempotencyKey != job.IdempotencyKey || !bytes.Equal(existing.Payload, job.Payload) { return Job{}, ErrIdempotencyConflict }
		return existing, nil
	}
	if job.MaxAttempts <= 0 { job.MaxAttempts = 3 }
	job.State = Queued
	s.jobs[job.ID] = job
	s.keys[job.IdempotencyKey] = job.ID
	return job, nil
}

func (s *Store) Get(id string) (Job, bool) { s.mu.Lock(); defer s.mu.Unlock(); j, ok := s.jobs[id]; return j, ok }
func (s *Store) Archive() []ArchiveEntry { s.mu.Lock(); defer s.mu.Unlock(); return append([]ArchiveEntry(nil), s.archive...) }

// Process claims one queued job. A failed attempt is requeued below the bound;
// poison jobs are archived and never become executable again.
func (s *Store) Process(ctx context.Context, id string, fn func(context.Context, Job) error) (Job, error) {
	s.mu.Lock()
	if s.stopped { s.mu.Unlock(); return Job{}, ErrStopped }
	job, ok := s.jobs[id]
	if !ok { s.mu.Unlock(); return Job{}, errors.New("job not found") }
	if job.State != Queued { s.mu.Unlock(); return job, nil }
	job.State, job.Attempts = Running, job.Attempts+1
	s.jobs[id] = job
	s.mu.Unlock()
	err := fn(ctx, job)
	s.mu.Lock(); defer s.mu.Unlock()
	if errors.Is(err, context.Canceled) { job.State = Cancelled; job.LastError = err.Error() } else if err == nil { job.State = Succeeded } else { job.LastError = err.Error(); if job.Attempts < job.MaxAttempts { job.State = Queued } else { job.State = Failed; s.archive = append(s.archive, ArchiveEntry{Job: job, Reason: "retry_exhausted"}) } }
	s.jobs[id] = job
	return job, err
}

func (s *Store) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	return nil
}
