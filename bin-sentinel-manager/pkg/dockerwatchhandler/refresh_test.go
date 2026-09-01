package dockerwatchhandler

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-sentinel-manager/models/asteriskaddress"
	"monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/cachehandler"
)

// freshTTL / staleTTL sit just inside and just outside the freshness window, so every test below
// exercises the real boundary rather than an arbitrary "obviously old" value.
var (
	freshTTL = asteriskaddress.TTL - asteriskaddress.FreshnessMargin
	staleTTL = asteriskaddress.TTL - asteriskaddress.FreshnessMargin - time.Nanosecond
)

// seededEntry describes one pre-existing state table entry for a refresh test.
type seededEntry struct {
	containerName string
	service       string
	ip            string
	asteriskID    string
}

func newRefreshTestHandler(t *testing.T, mc *gomock.Controller, entries []seededEntry) (*dockerWatchHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &dockerWatchHandler{
		cacheHandler: mockCache,
		state:        newStateTable(),
		flap:         newFlapTracker(flapWindow, flapThreshold),
	}

	for _, e := range entries {
		h.state.Create(e.containerName, e.service, e.ip, time.Now())
		if e.asteriskID != "" {
			h.state.Resolve(e.containerName, e.asteriskID)
		}
	}

	return h, mockCache
}

func Test_refreshOnce(t *testing.T) {
	tests := []struct {
		name string

		entries   []seededEntry
		addresses []*asteriskaddress.AsteriskAddress

		expectIDs map[string]string
	}{
		{
			name: "resolves_a_fresh_candidate",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "resolves_several_containers_independently",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
				{"voip-asterisk-call-docker-2", container.ServiceAsteriskCall, "172.24.0.102", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "72:ce:24:e6:51:2f", Address: "172.24.0.102", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{
				"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32",
				"voip-asterisk-call-docker-2": "72:ce:24:e6:51:2f",
			},
		},
		{
			name: "exactly_on_the_freshness_boundary_resolves",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: freshTTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "one_nanosecond_past_the_boundary_does_not_resolve",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: staleTTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": ""},
		},
		{
			name: "a_stale_key_never_clears_a_resolved_id",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: staleTTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "an_empty_scan_never_clears_a_resolved_id",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			addresses: []*asteriskaddress.AsteriskAddress{},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "a_scan_with_no_key_for_this_ip_never_clears_a_resolved_id",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "72:ce:24:e6:51:2f", Address: "172.24.0.199", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "a_dead_generation_key_for_the_same_ip_is_filtered_out",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				// the dead generation's key still exists for hours, but has not been refreshed.
				{ID: "de:ad:de:ad:de:ad", Address: "172.24.0.101", TTL: 12 * time.Hour},
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "an_id_bound_to_another_live_container_is_excluded",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
				// docker-2 was recreated onto the SAME static ip; docker-1 still holds the id.
				{"voip-asterisk-call-docker-2", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "72:ce:24:e6:51:2f", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{
				"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32",
				"voip-asterisk-call-docker-2": "72:ce:24:e6:51:2f",
			},
		},
		{
			name: "two_unbound_fresh_candidates_for_one_ip_stay_ambiguous",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "72:ce:24:e6:51:2f", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": ""},
		},
		{
			name: "an_ambiguous_pass_never_clears_an_already_resolved_id",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
				{"voip-asterisk-call-docker-2", container.ServiceAsteriskCall, "172.24.0.102", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				// two unbound fresh candidates for docker-2's ip.
				{ID: "aa:aa:aa:aa:aa:aa", Address: "172.24.0.102", TTL: asteriskaddress.TTL},
				{ID: "bb:bb:bb:bb:bb:bb", Address: "172.24.0.102", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{
				"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32",
				"voip-asterisk-call-docker-2": "",
			},
		},
		{
			name: "an_entry_with_no_ip_stays_unresolved",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": ""},
		},
		{
			name: "an_address_with_an_empty_value_is_ignored",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": ""},
		},
		{
			name: "a_nil_address_entry_is_skipped_without_panicking",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				nil,
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "a_different_fresh_id_does_not_overwrite_a_resolved_one",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "ff:ff:ff:ff:ff:ff", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "a_re_resolution_to_the_same_id_is_stable",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockCache := newRefreshTestHandler(t, mc, tt.entries)

			ctx := context.Background()
			mockCache.EXPECT().AsteriskAddressInternalScan(ctx).Return(tt.addresses, nil)

			if err := h.refreshOnce(ctx); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			for containerName, expectID := range tt.expectIDs {
				entry, ok := h.state.Get(containerName)
				if !ok {
					t.Fatalf("Wrong match. expect: entry %s exists, got: missing", containerName)
				}
				if entry.AsteriskID != expectID {
					t.Errorf("Wrong match for %s. expect: %q, got: %q", containerName, expectID, entry.AsteriskID)
				}
			}
		})
	}
}

// Test_refreshOnce_scanErrorKeepsResolvedIDs pins the other half of "freshness gates learning,
// never forgetting": a Redis failure must not be mistaken for "there are no fresh candidates".
func Test_refreshOnce_scanErrorKeepsResolvedIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockCache := newRefreshTestHandler(t, mc, []seededEntry{
		{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
	})

	ctx := context.Background()
	mockCache.EXPECT().AsteriskAddressInternalScan(ctx).Return(nil, errors.New("redis is down"))

	if err := h.refreshOnce(ctx); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	entry, ok := h.state.Get("voip-asterisk-call-docker-1")
	if !ok {
		t.Fatalf("Wrong match. expect: entry exists, got: missing")
	}
	if entry.AsteriskID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %q", entry.AsteriskID)
	}
}

// Test_refreshOnce_emptyTableSkipsScan pins that an idle sentinel does not hammer Redis. The
// mock's strict expectations make an unexpected call a failure.
func Test_refreshOnce_emptyTableSkipsScan(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newRefreshTestHandler(t, mc, nil)

	if err := h.refreshOnce(context.Background()); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_refreshOnce_repeatedStalePassesNeverDegrade simulates the exact failure mode design review
// round 3 flagged: many consecutive passes with nothing fresh must leave the resolved id intact
// forever, not erode it.
func Test_refreshOnce_repeatedStalePassesNeverDegrade(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockCache := newRefreshTestHandler(t, mc, []seededEntry{
		{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
	})

	ctx := context.Background()
	mockCache.EXPECT().AsteriskAddressInternalScan(ctx).Return([]*asteriskaddress.AsteriskAddress{
		{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: staleTTL},
	}, nil).Times(10)

	for i := 0; i < 10; i++ {
		if err := h.refreshOnce(ctx); err != nil {
			t.Fatalf("Wrong match at pass %d. expect: ok, got: %v", i, err)
		}
	}

	entry, _ := h.state.Get("voip-asterisk-call-docker-1")
	if entry.AsteriskID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %q", entry.AsteriskID)
	}
}

func Test_freshCandidatesByIP(t *testing.T) {
	tests := []struct {
		name string

		addresses []*asteriskaddress.AsteriskAddress

		expect map[string][]string
	}{
		{
			name: "groups_fresh_by_ip",

			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "a", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "b", Address: "172.24.0.101", TTL: freshTTL},
				{ID: "c", Address: "172.24.0.102", TTL: asteriskaddress.TTL},
			},

			expect: map[string][]string{
				"172.24.0.101": {"a", "b"},
				"172.24.0.102": {"c"},
			},
		},
		{
			name: "drops_stale",

			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "a", Address: "172.24.0.101", TTL: staleTTL},
			},

			expect: map[string][]string{},
		},
		{
			name: "drops_empty_id_and_empty_address_and_nil",

			addresses: []*asteriskaddress.AsteriskAddress{
				nil,
				{ID: "", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "a", Address: "", TTL: asteriskaddress.TTL},
			},

			expect: map[string][]string{},
		},
		{
			name: "empty_input",

			addresses: nil,

			expect: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := freshCandidatesByIP(tt.addresses)

			if len(res) != len(tt.expect) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
			for ip, expectIDs := range tt.expect {
				resIDs := res[ip]
				if len(resIDs) != len(expectIDs) {
					t.Fatalf("Wrong match for %s. expect: %v, got: %v", ip, expectIDs, resIDs)
				}
				for i := range expectIDs {
					if resIDs[i] != expectIDs[i] {
						t.Errorf("Wrong match for %s. expect: %v, got: %v", ip, expectIDs, resIDs)
					}
				}
			}
		})
	}
}

func Test_boundAsteriskIDs(t *testing.T) {
	entries := []*containerState{
		{ContainerName: "voip-asterisk-call-docker-1", AsteriskID: "3e:50:6b:43:bb:32"},
		{ContainerName: "voip-asterisk-call-docker-2", AsteriskID: ""},
		{ContainerName: "voip-asterisk-conference-docker-1", AsteriskID: "72:ce:24:e6:51:2f"},
	}

	res := boundAsteriskIDs(entries)

	if len(res) != 2 {
		t.Fatalf("Wrong match. expect: 2 bound ids, got: %d (%v)", len(res), res)
	}
	if res["3e:50:6b:43:bb:32"] != "voip-asterisk-call-docker-1" {
		t.Errorf("Wrong match. expect: voip-asterisk-call-docker-1, got: %s", res["3e:50:6b:43:bb:32"])
	}
	if res["72:ce:24:e6:51:2f"] != "voip-asterisk-conference-docker-1" {
		t.Errorf("Wrong match. expect: voip-asterisk-conference-docker-1, got: %s", res["72:ce:24:e6:51:2f"])
	}
	if _, ok := res[""]; ok {
		t.Errorf("Wrong match. expect: unresolved entries excluded, got: an empty-id binding")
	}
}

func Test_selectCandidates(t *testing.T) {
	entry := &containerState{ContainerName: "voip-asterisk-call-docker-1", AsteriskID: "own"}

	tests := []struct {
		name string

		freshIDs []string
		boundIDs map[string]string

		expect []string
	}{
		{
			name: "keeps_unbound_ids",

			freshIDs: []string{"a", "b"},
			boundIDs: map[string]string{},

			expect: []string{"a", "b"},
		},
		{
			name: "keeps_its_own_id",

			freshIDs: []string{"own"},
			boundIDs: map[string]string{"own": "voip-asterisk-call-docker-1"},

			expect: []string{"own"},
		},
		{
			name: "excludes_ids_bound_to_another_container",

			freshIDs: []string{"own", "other"},
			boundIDs: map[string]string{"own": "voip-asterisk-call-docker-1", "other": "voip-asterisk-call-docker-2"},

			expect: []string{"own"},
		},
		{
			name: "empty_input",

			freshIDs: nil,
			boundIDs: map[string]string{},

			expect: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := selectCandidates(tt.freshIDs, entry, tt.boundIDs)

			if len(res) != len(tt.expect) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
			for i := range tt.expect {
				if res[i] != tt.expect[i] {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
				}
			}
		})
	}
}

// Test_runRefreshLoop_stopsOnContextCancel pins the loop's shutdown path.
func Test_runRefreshLoop_stopsOnContextCancel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockCache := newRefreshTestHandler(t, mc, []seededEntry{
		{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
	})
	h.refreshInterval = time.Millisecond

	mockCache.EXPECT().AsteriskAddressInternalScan(gomock.Any()).Return([]*asteriskaddress.AsteriskAddress{
		{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
	}, nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		h.runRefreshLoop(ctx)
		close(done)
	}()

	// let at least one tick land, then shut down.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Wrong match. expect: the refresh loop to stop on context cancel, got: still running")
	}

	entry, _ := h.state.Get("voip-asterisk-call-docker-1")
	if entry.AsteriskID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %q", entry.AsteriskID)
	}
}

// Test_runRefreshLoop_survivesScanErrors pins that a failing pass does not kill the loop.
func Test_runRefreshLoop_survivesScanErrors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockCache := newRefreshTestHandler(t, mc, []seededEntry{
		{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
	})
	h.refreshInterval = time.Millisecond

	mockCache.EXPECT().AsteriskAddressInternalScan(gomock.Any()).Return(nil, errors.New("redis is down")).MinTimes(2)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		h.runRefreshLoop(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Wrong match. expect: the refresh loop to stop on context cancel, got: still running")
	}

	entry, _ := h.state.Get("voip-asterisk-call-docker-1")
	if entry.AsteriskID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %q", entry.AsteriskID)
	}
}

// Test_refreshOnce_idChangeKeepsTheExistingID pins the conservative resolution of the one branch
// the design's invariant says should never fire.
//
// The asterisk-id derives from the container's MAC, which is fixed for that container object's
// whole lifetime, and one table entry spans exactly one container generation. So a fresh candidate
// whose id DIFFERS from an entry's already-resolved id is either a real anomaly (a second
// container claiming this IP) or a latent bug -- in neither case is the new value more trustworthy
// than the old one.
//
// Keeping the old id is the conservative choice: it was resolved while this generation was
// demonstrably alive, whereas adopting an unexplained new one risks firing recovery against a
// DIFFERENT, still-live instance and redialing channels that never dropped. The WARN log is what
// makes "keep" observable rather than merely silent.
func Test_refreshOnce_idChangeKeepsTheExistingID(t *testing.T) {
	tests := []struct {
		name string

		entries   []seededEntry
		addresses []*asteriskaddress.AsteriskAddress

		expectIDs map[string]string
	}{
		{
			name: "single_conflicting_candidate",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "ff:ff:ff:ff:ff:ff", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "conflicting_candidate_at_the_freshness_boundary",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "ff:ff:ff:ff:ff:ff", Address: "172.24.0.101", TTL: freshTTL},
			},

			expectIDs: map[string]string{"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32"},
		},
		{
			name: "the_conflict_does_not_disturb_a_healthy_sibling",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
				{"voip-asterisk-call-docker-2", container.ServiceAsteriskCall, "172.24.0.102", ""},
			},
			addresses: []*asteriskaddress.AsteriskAddress{
				{ID: "ff:ff:ff:ff:ff:ff", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "72:ce:24:e6:51:2f", Address: "172.24.0.102", TTL: asteriskaddress.TTL},
			},

			expectIDs: map[string]string{
				"voip-asterisk-call-docker-1": "3e:50:6b:43:bb:32",
				"voip-asterisk-call-docker-2": "72:ce:24:e6:51:2f",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockCache := newRefreshTestHandler(t, mc, tt.entries)

			ctx := context.Background()
			mockCache.EXPECT().AsteriskAddressInternalScan(ctx).Return(tt.addresses, nil)

			if err := h.refreshOnce(ctx); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			for containerName, expectID := range tt.expectIDs {
				entry, ok := h.state.Get(containerName)
				if !ok {
					t.Fatalf("Wrong match. expect: entry %s exists, got: missing", containerName)
				}
				if entry.AsteriskID != expectID {
					t.Errorf("Wrong match for %s. expect: %q, got: %q", containerName, expectID, entry.AsteriskID)
				}
			}
		})
	}
}

// Test_refreshOnce_idChangeIsStableAcrossPasses pins that the conflicting candidate is rejected
// EVERY pass, not just the first -- a repeated anomaly must not eventually wear the old id down.
func Test_refreshOnce_idChangeIsStableAcrossPasses(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockCache := newRefreshTestHandler(t, mc, []seededEntry{
		{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
	})

	ctx := context.Background()
	mockCache.EXPECT().AsteriskAddressInternalScan(ctx).Return([]*asteriskaddress.AsteriskAddress{
		{ID: "ff:ff:ff:ff:ff:ff", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
	}, nil).Times(10)

	for i := 0; i < 10; i++ {
		if err := h.refreshOnce(ctx); err != nil {
			t.Fatalf("Wrong match at pass %d. expect: ok, got: %v", i, err)
		}
	}

	entry, _ := h.state.Get("voip-asterisk-call-docker-1")
	if entry.AsteriskID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %q", entry.AsteriskID)
	}
}
