package events

import (
	"context"
	"errors"
	"sync"
)

var ErrDuplicate = errors.New("duplicate event")
type Event struct { ID string; Subject string; Data []byte }
type MemoryBus struct { mu sync.Mutex; subjects map[string][]Event; seen map[string]bool; unavailable bool }
func NewMemoryBus() *MemoryBus { return &MemoryBus{subjects:map[string][]Event{},seen:map[string]bool{}} }
func (b *MemoryBus) Publish(_ context.Context, event Event) error { b.mu.Lock(); defer b.mu.Unlock(); if b.unavailable{return errors.New("event bus unavailable")}; if b.seen[event.ID]{return ErrDuplicate}; b.seen[event.ID]=true; event.Data=append([]byte(nil),event.Data...); b.subjects[event.Subject]=append(b.subjects[event.Subject],event); return nil }
func (b *MemoryBus) Consume(_ context.Context, subject string) []Event { b.mu.Lock(); defer b.mu.Unlock(); events:=append([]Event(nil),b.subjects[subject]...); b.subjects[subject]=nil; return events }
func (b *MemoryBus) SetUnavailable(value bool) { b.mu.Lock(); b.unavailable=value; b.mu.Unlock() }
