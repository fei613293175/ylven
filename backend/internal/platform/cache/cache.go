package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrRateLimited = errors.New("RATE_LIMITED")
var ErrUnavailable = errors.New("cache unavailable")
type entry struct { value []byte; expires time.Time }
type Memory struct { mu sync.Mutex; values map[string]entry; unavailable bool }
func NewMemory() *Memory { return &Memory{values:map[string]entry{}} }
func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration, now time.Time) error { m.mu.Lock(); defer m.mu.Unlock(); if m.unavailable{return ErrUnavailable}; m.values[key]=entry{value:append([]byte(nil),value...),expires:now.Add(ttl)}; return nil }
func (m *Memory) Get(_ context.Context, key string, now time.Time) ([]byte,bool,error) { m.mu.Lock(); defer m.mu.Unlock(); if m.unavailable{return nil,false,ErrUnavailable}; item,ok:=m.values[key]; if !ok||!item.expires.After(now){delete(m.values,key);return nil,false,nil}; return append([]byte(nil),item.value...),true,nil }
func (m *Memory) SetUnavailable(value bool) { m.mu.Lock(); m.unavailable=value; m.mu.Unlock() }

type Limiter struct { mu sync.Mutex; windows map[string]window; limit int; period time.Duration }
type window struct { started time.Time; count int }
func NewLimiter(limit int, period time.Duration) *Limiter { return &Limiter{windows:map[string]window{},limit:limit,period:period} }
func (l *Limiter) Allow(key string, now time.Time) error { l.mu.Lock(); defer l.mu.Unlock(); current:=l.windows[key]; if current.started.IsZero()||!now.Before(current.started.Add(l.period)){current=window{started:now}}; if current.count>=l.limit {l.windows[key]=current; return ErrRateLimited}; current.count++; l.windows[key]=current; return nil }
