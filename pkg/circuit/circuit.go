package circuit

import (
	"fmt"
	"sync"
	"time"
)

// State tracks one circuit's lifecycle (stub toward native manager).
type State int

const (
	Building State = iota
	Open
	Dirty
	Closed
)

// MaxCircuitDirtiness caps reuse of a circuit for new streams,
// mirroring torrc MaxCircuitDirtiness (seconds). After this age
// the manager treats the circuit as expired and ready to rotate.
const MaxCircuitDirtiness = 300 * time.Second

// Circuit is a minimal record; full crypto/onion layers come later.
type Circuit struct {
	ID       uint32
	State    State
	Created  time.Time
	LastUsed time.Time
	Hops     []string
}

// Manager keeps circuits; currently in-memory only.
type Manager struct {
	mu   sync.Mutex
	next uint32
	all  map[uint32]*Circuit
}

// New returns an empty manager.
func New() *Manager { return &Manager{all: map[uint32]*Circuit{}} }

// Build creates a placeholder circuit (no crypto yet).
func (m *Manager) Build(hops []string) *Circuit {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	now := time.Now()
	c := &Circuit{ID: m.next, State: Building, Created: now, LastUsed: now, Hops: hops}
	m.all[c.ID] = c
	return c
}

// MarkOpen moves a circuit to Open.
func (m *Manager) MarkOpen(id uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.all[id]
	if !ok {
		return fmt.Errorf("circuit %d not found", id)
	}
	c.State = Open
	c.LastUsed = time.Now()
	return nil
}

// MarkDirty flags a circuit dirty (no longer used for new streams).
func (m *Manager) MarkDirty(id uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.all[id]
	if !ok {
		return fmt.Errorf("circuit %d not found", id)
	}
	c.State = Dirty
	return nil
}

// Close marks a circuit closed.
func (m *Manager) Close(id uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.all[id]
	if !ok {
		return fmt.Errorf("circuit %d not found", id)
	}
	c.State = Closed
	return nil
}

// Touch refreshes LastUsed (stream activity).
func (m *Manager) Touch(id uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.all[id]
	if !ok {
		return fmt.Errorf("circuit %d not found", id)
	}
	c.LastUsed = time.Now()
	return nil
}

// Get returns a copy of one circuit.
func (m *Manager) Get(id uint32) (Circuit, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.all[id]
	if !ok {
		return Circuit{}, false
	}
	return *c, true
}

// Count returns the number of tracked circuits.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.all)
}

// IsExpired reports whether now is past Created+MaxCircuitDirtiness.
func IsExpired(c Circuit, now time.Time) bool {
	if c.Created.IsZero() {
		return false
	}
	return now.Sub(c.Created) > MaxCircuitDirtiness
}

// Expired returns a snapshot of circuits older than MaxCircuitDirtiness.
func (m *Manager) Expired(now time.Time) []Circuit {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Circuit
	for _, c := range m.all {
		if IsExpired(*c, now) {
			out = append(out, *c)
		}
	}
	return out
}

// PruneExpired drops expired circuits and returns how many were removed.
func (m *Manager) PruneExpired(now time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, c := range m.all {
		if IsExpired(*c, now) {
			delete(m.all, id)
			n++
		}
	}
	return n
}

// List returns a snapshot.
func (m *Manager) List() []Circuit {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Circuit, 0, len(m.all))
	for _, c := range m.all {
		out = append(out, *c)
	}
	return out
}
