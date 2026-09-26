package circuit

import (
	"testing"
	"time"
)

func TestMaxCircuitDirtinessValue(t *testing.T) {
	if MaxCircuitDirtiness != 300*time.Second {
		t.Fatalf("MaxCircuitDirtiness = %v, want 300s", MaxCircuitDirtiness)
	}
}

func TestBuildMarkOpenList(t *testing.T) {
	m := New()
	c := m.Build([]string{"a", "b"})
	if c.ID == 0 {
		t.Fatalf("expected non-zero ID")
	}
	if c.State != Building {
		t.Fatalf("state = %v, want Building", c.State)
	}
	if err := m.MarkOpen(c.ID); err != nil {
		t.Fatalf("MarkOpen: %v", err)
	}
	got, ok := m.Get(c.ID)
	if !ok {
		t.Fatalf("Get missing")
	}
	if got.State != Open {
		t.Fatalf("state = %v, want Open", got.State)
	}
	if len(m.List()) != 1 {
		t.Fatalf("List len = %d, want 1", len(m.List()))
	}
	if err := m.MarkOpen(9999); err == nil {
		t.Fatalf("expected error for unknown circuit")
	}
}

func TestIsExpiredBoundary(t *testing.T) {
	base := time.Now()
	fresh := Circuit{ID: 1, Created: base}
	if IsExpired(fresh, base.Add(MaxCircuitDirtiness)) {
		t.Fatalf("exactly at dirtiness must not be expired (strict >)")
	}
	if IsExpired(fresh, base.Add(MaxCircuitDirtiness-time.Second)) {
		t.Fatalf("before dirtiness must not be expired")
	}
	if !IsExpired(fresh, base.Add(MaxCircuitDirtiness+time.Second)) {
		t.Fatalf("after dirtiness must be expired")
	}
	if IsExpired(Circuit{}, base) {
		t.Fatalf("zero Created must not be expired")
	}
}

func TestPruneExpired(t *testing.T) {
	m := New()
	now := time.Now()
	old := m.Build([]string{"old"})
	fresh := m.Build([]string{"fresh"})
	// Age the old circuit past dirtiness.
	m.mu.Lock()
	m.all[old.ID].Created = now.Add(-MaxCircuitDirtiness - time.Minute)
	m.all[old.ID].LastUsed = now.Add(-MaxCircuitDirtiness - time.Minute)
	m.all[fresh.ID].Created = now
	m.all[fresh.ID].LastUsed = now
	m.mu.Unlock()

	expired := m.Expired(now)
	if len(expired) != 1 || expired[0].ID != old.ID {
		t.Fatalf("Expired = %v, want [old %d]", expired, old.ID)
	}
	if n := m.PruneExpired(now); n != 1 {
		t.Fatalf("PruneExpired = %d, want 1", n)
	}
	if m.Count() != 1 {
		t.Fatalf("Count = %d, want 1", m.Count())
	}
	if _, ok := m.Get(old.ID); ok {
		t.Fatalf("old circuit should be pruned")
	}
}

func TestMarkDirtyCloseTouch(t *testing.T) {
	m := New()
	c := m.Build(nil)
	if err := m.Touch(c.ID); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	if err := m.MarkDirty(c.ID); err != nil {
		t.Fatalf("MarkDirty: %v", err)
	}
	got, _ := m.Get(c.ID)
	if got.State != Dirty {
		t.Fatalf("state = %v, want Dirty", got.State)
	}
	if err := m.Close(c.ID); err != nil {
		t.Fatalf("Close: %v", err)
	}
	got, _ = m.Get(c.ID)
	if got.State != Closed {
		t.Fatalf("state = %v, want Closed", got.State)
	}
	for _, fn := range []func(uint32) error{m.MarkDirty, m.Close, m.Touch} {
		if err := fn(9999); err == nil {
			t.Fatalf("expected error for unknown circuit")
		}
	}
}
