package dockerwatchhandler

import (
	"sort"
	"sync"
	"time"
)

// containerState is one watched container GENERATION's last known state.
//
// "Generation" is load-bearing: an entry is created when a container starts (or is found running
// at sentinel's own boot) and destroyed when it dies. It never spans two container objects, which
// is what makes the sticky-last-known rule in refresh.go correct by invariant rather than by
// heuristic -- the asterisk-id derives from the container's MAC, which is fixed for that
// container object's entire lifetime, so the true id is CONSTANT over an entry's life.
type containerState struct {
	// ContainerName is the table key. It is the Compose-generated name, e.g.
	// "voip-asterisk-call-docker-2".
	ContainerName string
	// Service is the logical Asterisk workload, e.g. "asterisk-call".
	Service string
	// IP is the container's internal address, resolved once by inspect at start/boot time.
	IP string
	// AsteriskID is the id resolved by the background refresh loop, or "" while still unresolved.
	AsteriskID string
	// ObservedAt is when this entry was created.
	ObservedAt time.Time
}

// clone returns a copy so callers can never mutate the live entry outside the lock.
func (h *containerState) clone() *containerState {
	c := *h
	return &c
}

// stateTable is the per-container-name state map of design §3.3.
//
// A single mutex guards the whole map rather than one mutex per name. The design asked for
// "a single goroutine or a mutex-per-name" so that a die's read+delete for generation N cannot
// interleave with generation N+1's start write; one mutex over the map gives that ordering
// guarantee too, and at this cardinality (~6-10 entries, one refresh pass every 10s) per-name
// locks would be premature optimization.
type stateTable struct {
	mu      sync.Mutex
	entries map[string]*containerState
}

func newStateTable() *stateTable {
	return &stateTable{
		entries: map[string]*containerState{},
	}
}

// Create inserts (or replaces) the entry for a container name, ALWAYS starting from an
// unresolved asterisk-id.
//
// Initialization is deliberately not sticky. Stickiness (Resolve, below) governs UPDATES to an
// existing entry within one generation; a fresh generation must start from nothing, or a
// same-name replacement container would inherit the previous generation's id and answer a future
// death with the wrong one.
func (h *stateTable) Create(containerName string, service string, ip string, observedAt time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries[containerName] = &containerState{
		ContainerName: containerName,
		Service:       service,
		IP:            ip,
		AsteriskID:    "",
		ObservedAt:    observedAt,
	}
}

// Resolve records a newly learned asterisk-id for an existing entry.
//
// It returns true when the entry existed and now holds the given id. An EMPTY id is rejected
// outright: this method is the only write path for AsteriskID, so refusing "" here is what
// structurally guarantees the sticky-last-known rule -- there is no code path, anywhere, that can
// regress a resolved id back to unresolved. A refresh pass that learns nothing simply does not
// call this.
func (h *stateTable) Resolve(containerName string, asteriskID string) bool {
	if asteriskID == "" {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.entries[containerName]
	if !ok {
		return false
	}

	entry.AsteriskID = asteriskID

	return true
}

// Get returns a COPY of the entry for a container name.
func (h *stateTable) Get(containerName string) (*containerState, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.entries[containerName]
	if !ok {
		return nil, false
	}

	return entry.clone(), true
}

// Delete removes the entry and returns a copy of what it held.
//
// The read and the delete are ONE critical section on purpose: a die handler must consume exactly
// the state its own generation left behind, with no window for the replacement container's start
// handler to write between the two.
func (h *stateTable) Delete(containerName string) (*containerState, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.entries[containerName]
	if !ok {
		return nil, false
	}

	delete(h.entries, containerName)

	return entry.clone(), true
}

// List returns copies of every entry, ordered by container name for deterministic iteration.
func (h *stateTable) List() []*containerState {
	h.mu.Lock()
	defer h.mu.Unlock()

	names := make([]string, 0, len(h.entries))
	for name := range h.entries {
		names = append(names, name)
	}
	sort.Strings(names)

	res := make([]*containerState, 0, len(names))
	for _, name := range names {
		res = append(res, h.entries[name].clone())
	}

	return res
}

// Len returns the number of tracked containers.
func (h *stateTable) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.entries)
}

// flapTracker damps a crash-looping container.
//
// It counts DEATHS per container name inside a sliding window. Past the threshold, further deaths
// in that window are suppressed from publishing: repeated recovery attempts against a container
// stuck in a crash-loop would spam Homer/PJSIP for channels that likely never established, and a
// flapping container is a symptom to alert on rather than to keep redialing against.
type flapTracker struct {
	mu sync.Mutex

	window    time.Duration
	threshold int

	deaths map[string][]time.Time
}

func newFlapTracker(window time.Duration, threshold int) *flapTracker {
	return &flapTracker{
		window:    window,
		threshold: threshold,
		deaths:    map[string][]time.Time{},
	}
}

// Record registers a death at time now and reports whether it is within the allowed rate.
//
// It returns true when the event should be published, false when it is the (threshold+1)-th or
// later death inside the window. The current death is always counted, so a container that keeps
// dying stays suppressed until the window drains.
func (h *flapTracker) Record(containerName string, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := now.Add(-h.window)

	kept := make([]time.Time, 0, len(h.deaths[containerName])+1)
	for _, t := range h.deaths[containerName] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	h.deaths[containerName] = kept

	return len(kept) <= h.threshold
}

// Forget drops a container's flap history. Nothing calls it on the hot path today; it exists so a
// future explicit "this container is intentionally being replaced" signal can reset the window
// without restarting the process.
func (h *flapTracker) Forget(containerName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.deaths, containerName)
}
