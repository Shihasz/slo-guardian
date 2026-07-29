package controller

import "testing"

func TestTrackerAvailabilityPercent(t *testing.T) {
	tracker := NewTracker()
	key := "default/test-policy"

	avail, count := tracker.Record(key, true)
	if avail != 100.0 || count != 1 {
		t.Errorf("after 1 success: got avail=%v count=%v, want avail=100 count=1", avail, count)
	}

	tracker.Record(key, false)
	avail, count = tracker.Record(key, true)
	if count != 3 {
		t.Errorf("expected count=3, got %v", count)
	}
	expected := (2.0 / 3.0) * 100.0
	if diff := avail - expected; diff > 0.001 || diff < -0.001 {
		t.Errorf("got avail=%v, want %v", avail, expected)
	}
}

func TestTrackerWindowEviction(t *testing.T) {
	tracker := NewTracker()
	key := "default/window-test"

	// Fill the window with 100 successes
	for range windowSize {
		tracker.Record(key, true)
	}
	avail, count := tracker.Record(key, false)
	if count != windowSize {
		t.Errorf("expected window to stay at %v after eviction, got %v", windowSize, count)
	}
	// 99 successes + 1 failure out of 100 = 99%
	expected := 99.0
	if diff := avail - expected; diff > 0.001 || diff < -0.001 {
		t.Errorf("got avail=%v, want %v after eviction", avail, expected)
	}
}

func TestTrackerIndependentKeys(t *testing.T) {
	tracker := NewTracker()
	tracker.Record("policy-a", true)
	tracker.Record("policy-b", false)

	availA, _ := tracker.Record("policy-a", true)
	availB, _ := tracker.Record("policy-b", false)

	if availA != 100.0 {
		t.Errorf("policy-a should be unaffected by policy-b, got %v", availA)
	}
	if availB != 0.0 {
		t.Errorf("policy-b should be unaffected by policy-a, got %v", availB)
	}
}
