package jobs

import (
	"context"
	"errors"
	"testing"
)

func TestIdempotencyAndBoundedRetryArchive(t *testing.T) {
	s := NewStore(); first, err := s.Enqueue(Job{ID:"j1", IdempotencyKey:"k1", MaxAttempts:2}); if err != nil { t.Fatal(err) }
	if duplicate, err := s.Enqueue(Job{ID:"j1", IdempotencyKey:"k1"}); err != nil || duplicate.ID != first.ID { t.Fatalf("duplicate enqueue: %#v %v", duplicate, err) }
	if _, err := s.Enqueue(Job{ID:"j2", IdempotencyKey:"k1"}); !errors.Is(err, ErrIdempotencyConflict) { t.Fatalf("expected conflict, got %v", err) }
	for i := 0; i < 2; i++ { _, _ = s.Process(context.Background(), "j1", func(context.Context, Job) error { return errors.New("poison") }) }
	j, _ := s.Get("j1"); if j.State != Failed || len(s.Archive()) != 1 || j.Attempts != 2 { t.Fatalf("unexpected terminal job: %#v", j) }
}

func TestCancellationIsTerminal(t *testing.T) {
	s := NewStore(); _, _ = s.Enqueue(Job{ID:"j1", IdempotencyKey:"k1"})
	_, _ = s.Process(context.Background(), "j1", func(context.Context, Job) error { return context.Canceled })
	j, _ := s.Get("j1"); if j.State != Cancelled { t.Fatalf("got %s", j.State) }
}

func TestShutdownStopsNewWork(t *testing.T) {
	s := NewStore()
	if err := s.Shutdown(context.Background()); err != nil { t.Fatal(err) }
	if _, err := s.Enqueue(Job{ID: "j1", IdempotencyKey: "k1"}); !errors.Is(err, ErrStopped) { t.Fatalf("got %v", err) }
}
