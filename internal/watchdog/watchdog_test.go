package watchdog

import (
	"testing"
	"time"
)

func TestMinAutorotateFloor(t *testing.T) {
	if MinAutorotate != 10*time.Second {
		t.Fatalf("MinAutorotate = %v, want 10s", MinAutorotate)
	}
}

func TestShouldRotateOff(t *testing.T) {
	now := time.Now()
	if shouldRotate(now.Add(-time.Hour), 0, now) {
		t.Fatalf("interval 0 must never rotate")
	}
	if shouldRotate(now.Add(-time.Hour), -time.Second, now) {
		t.Fatalf("negative interval must never rotate")
	}
}

func TestShouldRotateTiming(t *testing.T) {
	now := time.Now()
	last := now.Add(-time.Hour)
	if !shouldRotate(last, 10*time.Minute, now) {
		t.Fatalf("overdue rotation should be due")
	}
	if shouldRotate(now.Add(-time.Minute), 10*time.Minute, now) {
		t.Fatalf("fresh rotation must not be due yet")
	}
	// Exact boundary counts as due (>=).
	if !shouldRotate(now.Add(-10*time.Minute), 10*time.Minute, now) {
		t.Fatalf("boundary rotation should be due")
	}
}

func TestShouldRotateNeverRotated(t *testing.T) {
	if !shouldRotate(time.Time{}, time.Minute, time.Now()) {
		t.Fatalf("zero last with active interval should be due")
	}
	if shouldRotate(time.Time{}, 0, time.Now()) {
		t.Fatalf("zero last with disabled interval must not rotate")
	}
}
