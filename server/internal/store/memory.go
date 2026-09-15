package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/neta"
)

type Binding struct {
	ID           string
	Session      string
	RefreshHint  string
	AccessToken  string
	RefreshToken string
	Meta         neta.VehicleMeta
	Snapshots    []neta.Snapshot
	Energy       neta.EnergyStat
	SyncStatus   string
	SyncError    string
	SyncedAt     clock.Instant
	Disabled     bool
}

type Memory struct {
	mu   sync.Mutex
	byID map[string]*Binding
}

func NewMemory() *Memory {
	return &Memory{byID: map[string]*Binding{}}
}

func (m *Memory) Put(b *Binding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b.ID == "" {
		b.ID = NewSessionID()
	}
	m.byID[b.Session] = b
	return nil
}

func (m *Memory) Get(session string) *Binding {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byID[session]
}

func (m *Memory) Delete(session string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, session)
}

func NewSessionID() string {
	var b [24]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func HintToken(token string) string {
	if len(token) < 8 {
		return "****"
	}
	return "****" + token[len(token)-4:]
}
