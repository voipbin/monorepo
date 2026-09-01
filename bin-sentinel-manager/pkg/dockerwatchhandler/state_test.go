package dockerwatchhandler

import (
	"sync"
	"testing"
	"time"

	"monorepo/bin-sentinel-manager/models/container"
)

func Test_stateTable_Create(t *testing.T) {
	observedAt := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		containerName string
		service       string
		ip            string

		expect *containerState
	}{
		{
			name: "asterisk_call",

			containerName: "voip-asterisk-call-docker-1",
			service:       container.ServiceAsteriskCall,
			ip:            "172.24.0.101",

			expect: &containerState{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       container.ServiceAsteriskCall,
				IP:            "172.24.0.101",
				AsteriskID:    "",
				ObservedAt:    observedAt,
			},
		},
		{
			name: "unresolvable_ip",

			containerName: "voip-asterisk-registrar-docker-2",
			service:       container.ServiceAsteriskRegistrar,
			ip:            "",

			expect: &containerState{
				ContainerName: "voip-asterisk-registrar-docker-2",
				Service:       container.ServiceAsteriskRegistrar,
				IP:            "",
				AsteriskID:    "",
				ObservedAt:    observedAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := newStateTable()
			table.Create(tt.containerName, tt.service, tt.ip, observedAt)

			res, ok := table.Get(tt.containerName)
			if !ok {
				t.Fatalf("Wrong match. expect: entry exists, got: missing")
			}
			if *res != *tt.expect {
				t.Errorf("Wrong match. expect: %+v, got: %+v", tt.expect, res)
			}
		})
	}
}

// Test_stateTable_CreateAlwaysResetsAsteriskID pins the initialization half of the design's
// sticky rule: stickiness governs UPDATES within one container generation, never initialization.
// A same-name REPLACEMENT container must not inherit the dead generation's id, or the next death
// would fire recovery against the wrong asterisk-id.
func Test_stateTable_CreateAlwaysResetsAsteriskID(t *testing.T) {
	table := newStateTable()
	name := "voip-asterisk-call-docker-1"

	table.Create(name, container.ServiceAsteriskCall, "172.24.0.101", time.Now())
	if !table.Resolve(name, "3e:50:6b:43:bb:32") {
		t.Fatalf("Wrong match. expect: resolve ok, got: not ok")
	}

	// generation N+1 reuses the same container name and the same static IP.
	table.Create(name, container.ServiceAsteriskCall, "172.24.0.101", time.Now())

	res, ok := table.Get(name)
	if !ok {
		t.Fatalf("Wrong match. expect: entry exists, got: missing")
	}
	if res.AsteriskID != "" {
		t.Errorf("Wrong match. expect: empty asterisk id on a new generation, got: %s", res.AsteriskID)
	}
}

// Test_stateTable_ResolveIsSticky is THE regression test for design §3.3's sticky-last-known
// rule. Resolve is the only write path for AsteriskID, and it must refuse "" -- otherwise a
// refresh pass that momentarily finds no fresh candidate would clear a correctly-resolved id and,
// combined with call-manager's empty-id guard, silently skip the exact recovery this service
// exists to trigger.
func Test_stateTable_ResolveIsSticky(t *testing.T) {
	tests := []struct {
		name string

		initialID string
		resolveID string

		expectResult bool
		expectID     string
	}{
		{
			name: "resolves_from_unresolved",

			initialID: "",
			resolveID: "3e:50:6b:43:bb:32",

			expectResult: true,
			expectID:     "3e:50:6b:43:bb:32",
		},
		{
			name: "empty_id_never_clears_a_resolved_entry",

			initialID: "3e:50:6b:43:bb:32",
			resolveID: "",

			expectResult: false,
			expectID:     "3e:50:6b:43:bb:32",
		},
		{
			name: "empty_id_on_an_unresolved_entry_is_a_no_op",

			initialID: "",
			resolveID: "",

			expectResult: false,
			expectID:     "",
		},
		{
			// Resolve is the low-level primitive and DOES overwrite. The policy that refuses an
			// unexplained id change for an already-resolved entry lives one level up, in
			// refreshOnce -- see Test_refreshOnce_idChangeKeepsTheExistingID.
			name: "the_primitive_itself_overwrites_a_different_id",

			initialID: "3e:50:6b:43:bb:32",
			resolveID: "72:ce:24:e6:51:2f",

			expectResult: true,
			expectID:     "72:ce:24:e6:51:2f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := newStateTable()
			name := "voip-asterisk-call-docker-1"
			table.Create(name, container.ServiceAsteriskCall, "172.24.0.101", time.Now())

			if tt.initialID != "" {
				table.Resolve(name, tt.initialID)
			}

			if res := table.Resolve(name, tt.resolveID); res != tt.expectResult {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResult, res)
			}

			entry, ok := table.Get(name)
			if !ok {
				t.Fatalf("Wrong match. expect: entry exists, got: missing")
			}
			if entry.AsteriskID != tt.expectID {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectID, entry.AsteriskID)
			}
		})
	}
}

func Test_stateTable_ResolveUnknownContainer(t *testing.T) {
	table := newStateTable()

	if res := table.Resolve("voip-asterisk-call-docker-9", "3e:50:6b:43:bb:32"); res {
		t.Errorf("Wrong match. expect: false for an unknown container, got: true")
	}
	if table.Len() != 0 {
		t.Errorf("Wrong match. expect: Resolve must not create entries, got: %d entries", table.Len())
	}
}

func Test_stateTable_Delete(t *testing.T) {
	table := newStateTable()
	name := "voip-asterisk-call-docker-1"

	table.Create(name, container.ServiceAsteriskCall, "172.24.0.101", time.Now())
	table.Resolve(name, "3e:50:6b:43:bb:32")

	res, ok := table.Delete(name)
	if !ok {
		t.Fatalf("Wrong match. expect: entry exists, got: missing")
	}
	if res.AsteriskID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %s", res.AsteriskID)
	}

	if _, ok := table.Get(name); ok {
		t.Errorf("Wrong match. expect: entry removed, got: still present")
	}

	// a second delete must report the entry as absent rather than returning stale data.
	if _, ok := table.Delete(name); ok {
		t.Errorf("Wrong match. expect: second delete reports missing, got: ok")
	}
}

func Test_stateTable_DeleteUnknownContainer(t *testing.T) {
	table := newStateTable()

	res, ok := table.Delete("voip-asterisk-call-docker-9")
	if ok {
		t.Errorf("Wrong match. expect: false, got: true")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %+v", res)
	}
}

// Test_stateTable_GetReturnsCopy pins that callers cannot mutate the live entry through a
// returned pointer -- a die handler holding a snapshot must not be able to corrupt the table.
func Test_stateTable_GetReturnsCopy(t *testing.T) {
	table := newStateTable()
	name := "voip-asterisk-call-docker-1"
	table.Create(name, container.ServiceAsteriskCall, "172.24.0.101", time.Now())
	table.Resolve(name, "3e:50:6b:43:bb:32")

	first, _ := table.Get(name)
	first.AsteriskID = "tampered"
	first.IP = "tampered"

	second, _ := table.Get(name)
	if second.AsteriskID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %s", second.AsteriskID)
	}
	if second.IP != "172.24.0.101" {
		t.Errorf("Wrong match. expect: 172.24.0.101, got: %s", second.IP)
	}
}

func Test_stateTable_ListIsSortedAndCopied(t *testing.T) {
	table := newStateTable()
	table.Create("voip-asterisk-registrar-docker-1", container.ServiceAsteriskRegistrar, "172.24.0.121", time.Now())
	table.Create("voip-asterisk-call-docker-2", container.ServiceAsteriskCall, "172.24.0.102", time.Now())
	table.Create("voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", time.Now())

	expect := []string{
		"voip-asterisk-call-docker-1",
		"voip-asterisk-call-docker-2",
		"voip-asterisk-registrar-docker-1",
	}

	res := table.List()
	if len(res) != len(expect) {
		t.Fatalf("Wrong match. expect: %d entries, got: %d", len(expect), len(res))
	}
	for i := range expect {
		if res[i].ContainerName != expect[i] {
			t.Fatalf("Wrong match. expect: %v, got entry %d: %s", expect, i, res[i].ContainerName)
		}
	}

	res[0].AsteriskID = "tampered"
	again, _ := table.Get("voip-asterisk-call-docker-1")
	if again.AsteriskID != "" {
		t.Errorf("Wrong match. expect: List returns copies, got mutated entry: %s", again.AsteriskID)
	}
}

func Test_stateTable_Len(t *testing.T) {
	table := newStateTable()

	if table.Len() != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", table.Len())
	}

	table.Create("voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", time.Now())
	table.Create("voip-asterisk-call-docker-2", container.ServiceAsteriskCall, "172.24.0.102", time.Now())
	if table.Len() != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", table.Len())
	}

	table.Delete("voip-asterisk-call-docker-1")
	if table.Len() != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", table.Len())
	}
}

// Test_stateTable_ConcurrentAccess exercises the lock under `go test -race`: the refresh loop and
// the event handler mutate the table from different goroutines, and design §3.3 requires that a
// die's read+delete cannot interleave with a start's write for the same name.
func Test_stateTable_ConcurrentAccess(t *testing.T) {
	table := newStateTable()
	names := []string{
		"voip-asterisk-call-docker-1",
		"voip-asterisk-call-docker-2",
		"voip-asterisk-conference-docker-1",
	}

	var wg sync.WaitGroup
	for _, name := range names {
		for i := 0; i < 50; i++ {
			wg.Add(4)

			go func() { defer wg.Done(); table.Create(name, container.ServiceAsteriskCall, "172.24.0.101", time.Now()) }()
			go func() { defer wg.Done(); table.Resolve(name, "3e:50:6b:43:bb:32") }()
			go func() { defer wg.Done(); table.Get(name) }()
			go func() { defer wg.Done(); table.List() }()
		}
	}
	wg.Wait()

	// nothing to assert beyond "no race, no panic"; the table's final content is
	// interleaving-dependent by construction.
	if table.Len() > len(names) {
		t.Errorf("Wrong match. expect: at most %d entries, got: %d", len(names), table.Len())
	}
}

func Test_flapTracker_Record(t *testing.T) {
	base := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	name := "voip-asterisk-call-docker-1"

	tests := []struct {
		name string

		window    time.Duration
		threshold int
		offsets   []time.Duration

		expect []bool
	}{
		{
			name: "under_the_threshold_all_publish",

			window:    60 * time.Second,
			threshold: 3,
			offsets:   []time.Duration{0, time.Second, 2 * time.Second},

			expect: []bool{true, true, true},
		},
		{
			name: "fourth_death_in_the_window_is_damped",

			window:    60 * time.Second,
			threshold: 3,
			offsets:   []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second},

			expect: []bool{true, true, true, false},
		},
		{
			name: "stays_damped_while_the_container_keeps_flapping",

			window:    60 * time.Second,
			threshold: 3,
			offsets:   []time.Duration{0, 1 * time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second, 5 * time.Second},

			expect: []bool{true, true, true, false, false, false},
		},
		{
			name: "window_drains_and_publishing_resumes",

			window:    60 * time.Second,
			threshold: 3,
			offsets:   []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second, 90 * time.Second},

			expect: []bool{true, true, true, false, true},
		},
		{
			name: "exactly_on_the_window_edge_is_expired",

			window:    60 * time.Second,
			threshold: 1,
			offsets:   []time.Duration{0, 60 * time.Second},

			expect: []bool{true, true},
		},
		{
			name: "one_nanosecond_inside_the_window_still_counts",

			window:    60 * time.Second,
			threshold: 1,
			offsets:   []time.Duration{0, 60*time.Second - time.Nanosecond},

			expect: []bool{true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := newFlapTracker(tt.window, tt.threshold)

			for i, offset := range tt.offsets {
				res := tracker.Record(name, base.Add(offset))
				if res != tt.expect[i] {
					t.Errorf("Wrong match at death %d. expect: %v, got: %v", i, tt.expect[i], res)
				}
			}
		})
	}
}

// Test_flapTracker_PerContainerIsolation pins that one flapping container does not damp a
// different, healthy one.
func Test_flapTracker_PerContainerIsolation(t *testing.T) {
	base := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	tracker := newFlapTracker(60*time.Second, flapThreshold)

	flapping := "voip-asterisk-call-docker-1"
	healthy := "voip-asterisk-call-docker-2"

	for i := 0; i < flapThreshold+2; i++ {
		tracker.Record(flapping, base.Add(time.Duration(i)*time.Second))
	}

	if res := tracker.Record(flapping, base.Add(10*time.Second)); res {
		t.Errorf("Wrong match. expect: flapping container damped, got: publish")
	}
	if res := tracker.Record(healthy, base.Add(10*time.Second)); !res {
		t.Errorf("Wrong match. expect: healthy container publishes, got: damped")
	}
}

func Test_flapTracker_ConcurrentAccess(t *testing.T) {
	tracker := newFlapTracker(60*time.Second, flapThreshold)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); tracker.Record("voip-asterisk-call-docker-1", time.Now()) }()
		go func() { defer wg.Done(); tracker.Record("voip-asterisk-call-docker-2", time.Now()) }()
	}
	wg.Wait()
}
