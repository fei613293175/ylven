package scheduling

import (
	"context"
	"testing"
	"time"
)

func TestContentionExpiryAndMonotonicFencing(t *testing.T) {
	now := time.Unix(100, 0); locks := NewLockStore()
	a, err := locks.Acquire("tick", "a", now, time.Second); if err != nil { t.Fatal(err) }
	if _, err := locks.Acquire("tick", "b", now.Add(500*time.Millisecond), time.Second); err != ErrContended { t.Fatalf("expected contention, got %v", err) }
	b, err := locks.Acquire("tick", "b", now.Add(time.Second), time.Second); if err != nil { t.Fatal(err) }
	if b.FencingToken <= a.FencingToken { t.Fatalf("fencing token did not increase") }
	if locks.Release(a) { t.Fatal("stale lease released current owner") }
	if !locks.Release(b) { t.Fatal("current lease did not release") }
}

func TestRunOnceExecutesOnlyWhenLeaseHeld(t *testing.T) {
	locks := NewLockStore(); s := Scheduler{Locks: locks, Owner:"a", TTL:time.Second}; calls := 0
	run, err := s.RunOnce(context.Background(), "tick", time.Unix(1, 0), func(context.Context, uint64) error { calls++; return nil }); if err != nil || !run || calls != 1 { t.Fatalf("first run: %v %v %d", run, err, calls) }
	lease, err := locks.Acquire("tick", "a", time.Unix(2, 0), time.Second); if err != nil { t.Fatal(err) }
	other := Scheduler{Locks: locks, Owner:"b", TTL:time.Second}; run, err = other.RunOnce(context.Background(), "tick", time.Unix(1, 0), func(context.Context, uint64) error { calls++; return nil }); if err != nil || run || calls != 1 { t.Fatalf("duplicate run: %v %v %d", run, err, calls) }
	if !locks.Release(lease) { t.Fatal("held lease did not release") }
}
