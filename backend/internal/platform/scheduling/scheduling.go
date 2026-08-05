package scheduling

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrContended = errors.New("SCHEDULER_LOCK_CONTENDED")

type Lease struct { Name string; Owner string; FencingToken uint64; ExpiresAt time.Time }
type LockStore struct { mu sync.Mutex; leases map[string]Lease; next map[string]uint64 }
func NewLockStore() *LockStore { return &LockStore{leases: map[string]Lease{}, next: map[string]uint64{}} }

func (s *LockStore) Acquire(name, owner string, now time.Time, ttl time.Duration) (Lease, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	if current, ok := s.leases[name]; ok && current.ExpiresAt.After(now) { return Lease{}, ErrContended }
	s.next[name]++
	lease := Lease{Name: name, Owner: owner, FencingToken: s.next[name], ExpiresAt: now.Add(ttl)}
	s.leases[name] = lease
	return lease, nil
}
func (s *LockStore) Release(lease Lease) bool { s.mu.Lock(); defer s.mu.Unlock(); current, ok := s.leases[lease.Name]; if !ok || current.Owner != lease.Owner || current.FencingToken != lease.FencingToken { return false }; delete(s.leases, lease.Name); return true }

type Scheduler struct { Locks *LockStore; Owner string; TTL time.Duration }
func (s Scheduler) RunOnce(ctx context.Context, name string, now time.Time, fn func(context.Context, uint64) error) (bool, error) {
	lease, err := s.Locks.Acquire(name, s.Owner, now, s.TTL)
	if err != nil { if errors.Is(err, ErrContended) { return false, nil }; return false, err }
	defer s.Locks.Release(lease)
	if err := fn(ctx, lease.FencingToken); err != nil { return true, err }
	return true, nil
}
