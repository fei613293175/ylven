package database

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

type Migration struct { Version int64; Name string; SQL string }
type AppliedMigration struct { Version int64; Name string; Checksum string; AppliedAt time.Time }
var ErrChecksumMismatch = errors.New("migration checksum mismatch")
var ErrDowngrade = errors.New("migration rollback is not supported")

type Migrator struct { mu sync.Mutex; applied map[int64]AppliedMigration }
func NewMigrator() *Migrator { return &Migrator{applied: map[int64]AppliedMigration{}} }
func Checksum(sql string) string { sum:=sha256.Sum256([]byte(sql)); return hex.EncodeToString(sum[:]) }
func (m *Migrator) Apply(list []Migration, now time.Time) ([]AppliedMigration, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	sort.Slice(list, func(i,j int) bool { return list[i].Version < list[j].Version })
	for _, migration := range list {
		hash := Checksum(migration.SQL)
		if existing, ok := m.applied[migration.Version]; ok && (existing.Name != migration.Name || existing.Checksum != hash) { return nil, ErrChecksumMismatch }
		if _, ok := m.applied[migration.Version]; !ok { m.applied[migration.Version] = AppliedMigration{Version:migration.Version, Name:migration.Name, Checksum:hash, AppliedAt:now} }
	}
	return m.List(), nil
}
func (m *Migrator) List() []AppliedMigration { result:=make([]AppliedMigration,0,len(m.applied)); for _, item:=range m.applied { result=append(result,item) }; sort.Slice(result,func(i,j int)bool{return result[i].Version<result[j].Version}); return result }
func (m *Migrator) Rollback(version int64) error { m.mu.Lock(); defer m.mu.Unlock(); if _, ok:=m.applied[version]; ok { return ErrDowngrade }; return nil }
