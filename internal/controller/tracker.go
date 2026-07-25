package controller

import "sync"

const windowSize = 100

// checkHistory holds a fixed-size sliding window of health check results
// for a single SLOPolicy, true = success, false = failure.
type checkHistory struct {
	results []bool
	pos     int
	filled  bool
}

func (h *checkHistory) record(success bool) {
	if h.results == nil {
		h.results = make([]bool, windowSize)
	}
	h.results[h.pos] = success
	h.pos = (h.pos + 1) % windowSize
	if h.pos == 0 {
		h.filled = true
	}
}

// availabilityPercent returns success rate over the window, and how many
// checks are currently counted in it.
func (h *checkHistory) availabilityPercent() (float64, int) {
	count := h.pos
	if h.filled {
		count = windowSize
	}
	if count == 0 {
		return 100.0, 0
	}
	successes := 0
	for i := 0; i < count; i++ {
		if h.results[i] {
			successes++
		}
	}
	return (float64(successes) / float64(count)) * 100.0, count
}

// Tracker holds check history per SLOPolicy, keyed by "namespace/name".
type Tracker struct {
	mu   sync.Mutex
	data map[string]*checkHistory
}

func NewTracker() *Tracker {
	return &Tracker{data: make(map[string]*checkHistory)}
}

func (t *Tracker) Record(key string, success bool) (float64, int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	h, ok := t.data[key]
	if !ok {
		h = &checkHistory{}
		t.data[key] = h
	}
	h.record(success)
	return h.availabilityPercent()
}
