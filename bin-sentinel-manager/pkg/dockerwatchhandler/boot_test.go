package dockerwatchhandler

import (
	"context"
	"testing"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-sentinel-manager/models/container"
)

// summaryOf builds a minimal container.Summary for the list response.
func summaryOf(id string, name string) dockercontainer.Summary {
	return dockercontainer.Summary{
		ID:    id,
		Names: []string{"/" + name},
	}
}

// inspectWithIP builds a minimal inspect response carrying one production-network address.
func inspectWithIP(ip string) dockercontainer.InspectResponse {
	return dockercontainer.InspectResponse{
		NetworkSettings: &dockercontainer.NetworkSettings{
			Networks: map[string]*dockernetwork.EndpointSettings{
				"production": {IPAddress: ip},
			},
		},
	}
}

// Test_bootReconcile is the regression test for design §3.3 step 0. Without boot-time
// reconciliation, every sentinel restart leaves the already-running Asterisk containers with no
// table entry -- an INDEFINITE blind spot until their next recreation, not a bounded one, since
// sentinel runs single-replica with no sibling to cover for it.
func Test_bootReconcile(t *testing.T) {
	tests := []struct {
		name string

		summaries []dockercontainer.Summary
		inspects  map[string]dockercontainer.InspectResponse

		expectEntries map[string]*containerState
	}{
		{
			name: "seeds_every_watched_running_container",

			summaries: []dockercontainer.Summary{
				summaryOf("id-call-1", "voip-asterisk-call-docker-1"),
				summaryOf("id-call-2", "voip-asterisk-call-docker-2"),
				summaryOf("id-conf-1", "voip-asterisk-conference-docker-1"),
				summaryOf("id-reg-1", "voip-asterisk-registrar-docker-1"),
			},
			inspects: map[string]dockercontainer.InspectResponse{
				"id-call-1": inspectWithIP("172.24.0.101"),
				"id-call-2": inspectWithIP("172.24.0.102"),
				"id-conf-1": inspectWithIP("172.24.0.111"),
				"id-reg-1":  inspectWithIP("172.24.0.121"),
			},

			expectEntries: map[string]*containerState{
				"voip-asterisk-call-docker-1":       {Service: container.ServiceAsteriskCall, IP: "172.24.0.101"},
				"voip-asterisk-call-docker-2":       {Service: container.ServiceAsteriskCall, IP: "172.24.0.102"},
				"voip-asterisk-conference-docker-1": {Service: container.ServiceAsteriskConference, IP: "172.24.0.111"},
				"voip-asterisk-registrar-docker-1":  {Service: container.ServiceAsteriskRegistrar, IP: "172.24.0.121"},
			},
		},
		{
			name: "ignores_unwatched_containers_and_proxy_sidecars",

			summaries: []dockercontainer.Summary{
				summaryOf("id-call-1", "voip-asterisk-call-docker-1"),
				summaryOf("id-sidecar", "voip-asterisk-call-docker-1-asterisk-call-proxy-1"),
				summaryOf("id-kamailio", "voip-kamailio-docker-1"),
				summaryOf("id-rtpengine", "voip-rtpengine-docker-1"),
				summaryOf("id-manager", "bin-call-manager-call-manager-1"),
			},
			inspects: map[string]dockercontainer.InspectResponse{
				"id-call-1": inspectWithIP("172.24.0.101"),
			},

			expectEntries: map[string]*containerState{
				"voip-asterisk-call-docker-1": {Service: container.ServiceAsteriskCall, IP: "172.24.0.101"},
			},
		},
		{
			name: "a_container_with_no_resolvable_ip_is_still_seeded",

			summaries: []dockercontainer.Summary{
				summaryOf("id-call-1", "voip-asterisk-call-docker-1"),
			},
			inspects: map[string]dockercontainer.InspectResponse{
				"id-call-1": {},
			},

			expectEntries: map[string]*containerState{
				"voip-asterisk-call-docker-1": {Service: container.ServiceAsteriskCall, IP: ""},
			},
		},
		{
			name: "a_summary_without_names_is_skipped",

			summaries: []dockercontainer.Summary{
				{ID: "id-nameless"},
				summaryOf("id-call-1", "voip-asterisk-call-docker-1"),
			},
			inspects: map[string]dockercontainer.InspectResponse{
				"id-call-1": inspectWithIP("172.24.0.101"),
			},

			expectEntries: map[string]*containerState{
				"voip-asterisk-call-docker-1": {Service: container.ServiceAsteriskCall, IP: "172.24.0.101"},
			},
		},
		{
			name: "nothing_running",

			summaries: []dockercontainer.Summary{},
			inspects:  map[string]dockercontainer.InspectResponse{},

			expectEntries: map[string]*containerState{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDocker := NewMockdockerClient(mc)
			h := &dockerWatchHandler{
				dockerClient: mockDocker,
				state:        newStateTable(),
				flap:         newFlapTracker(flapWindow, flapThreshold),
			}

			ctx := context.Background()
			mockDocker.EXPECT().ContainerList(ctx, dockercontainer.ListOptions{}).Return(tt.summaries, nil)
			for id, inspect := range tt.inspects {
				mockDocker.EXPECT().ContainerInspect(ctx, id).Return(inspect, nil)
			}

			if err := h.bootReconcile(ctx); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if h.state.Len() != len(tt.expectEntries) {
				t.Fatalf("Wrong match. expect: %d entries, got: %d (%v)", len(tt.expectEntries), h.state.Len(), h.state.List())
			}

			for containerName, expect := range tt.expectEntries {
				entry, ok := h.state.Get(containerName)
				if !ok {
					t.Fatalf("Wrong match. expect: entry %s exists, got: missing", containerName)
				}
				if entry.Service != expect.Service {
					t.Errorf("Wrong match for %s. expect: %s, got: %s", containerName, expect.Service, entry.Service)
				}
				if entry.IP != expect.IP {
					t.Errorf("Wrong match for %s. expect: %s, got: %s", containerName, expect.IP, entry.IP)
				}
				// step 0 always seeds an UNRESOLVED id; the immediate refresh pass resolves it.
				if entry.AsteriskID != "" {
					t.Errorf("Wrong match for %s. expect: empty asterisk id at seed time, got: %s", containerName, entry.AsteriskID)
				}
			}
		})
	}
}

// Test_bootReconcile_listErrorFailsLoud pins the fail-loud requirement of design §3.2: an
// unreachable docker-socket-proxy must crash the process, not leave sentinel silently watching
// nothing.
func Test_bootReconcile_listErrorFailsLoud(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	h := &dockerWatchHandler{
		dockerClient: mockDocker,
		state:        newStateTable(),
		flap:         newFlapTracker(flapWindow, flapThreshold),
	}

	ctx := context.Background()
	mockDocker.EXPECT().ContainerList(ctx, dockercontainer.ListOptions{}).Return(nil, errors.New("connection refused"))

	if err := h.bootReconcile(ctx); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
}

// Test_bootReconcile_inspectErrorSkipsOneContainer pins that ONE unreadable container does not
// abort the whole reconciliation -- the remaining containers are still worth seeding.
func Test_bootReconcile_inspectErrorSkipsOneContainer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	h := &dockerWatchHandler{
		dockerClient: mockDocker,
		state:        newStateTable(),
		flap:         newFlapTracker(flapWindow, flapThreshold),
	}

	ctx := context.Background()
	mockDocker.EXPECT().ContainerList(ctx, dockercontainer.ListOptions{}).Return([]dockercontainer.Summary{
		summaryOf("id-call-1", "voip-asterisk-call-docker-1"),
		summaryOf("id-call-2", "voip-asterisk-call-docker-2"),
	}, nil)
	mockDocker.EXPECT().ContainerInspect(ctx, "id-call-1").Return(dockercontainer.InspectResponse{}, errors.New("no such container"))
	mockDocker.EXPECT().ContainerInspect(ctx, "id-call-2").Return(inspectWithIP("172.24.0.102"), nil)

	if err := h.bootReconcile(ctx); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if _, ok := h.state.Get("voip-asterisk-call-docker-1"); ok {
		t.Errorf("Wrong match. expect: the uninspectable container to be skipped, got: seeded")
	}

	entry, ok := h.state.Get("voip-asterisk-call-docker-2")
	if !ok {
		t.Fatalf("Wrong match. expect: voip-asterisk-call-docker-2 seeded, got: missing")
	}
	if entry.IP != "172.24.0.102" {
		t.Errorf("Wrong match. expect: 172.24.0.102, got: %s", entry.IP)
	}
}

// Test_bootReconcile_replacesStaleEntries pins that a second reconciliation (a re-run, or a
// re-entry after a restart) overwrites rather than merges -- a stale entry must never answer a
// future death with the wrong generation's data.
func Test_bootReconcile_replacesStaleEntries(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	h := &dockerWatchHandler{
		dockerClient: mockDocker,
		state:        newStateTable(),
		flap:         newFlapTracker(flapWindow, flapThreshold),
	}

	ctx := context.Background()
	mockDocker.EXPECT().ContainerList(ctx, dockercontainer.ListOptions{}).Return([]dockercontainer.Summary{
		summaryOf("id-call-1", "voip-asterisk-call-docker-1"),
	}, nil).Times(2)
	mockDocker.EXPECT().ContainerInspect(ctx, "id-call-1").Return(inspectWithIP("172.24.0.101"), nil)
	mockDocker.EXPECT().ContainerInspect(ctx, "id-call-1").Return(inspectWithIP("172.24.0.109"), nil)

	if err := h.bootReconcile(ctx); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	h.state.Resolve("voip-asterisk-call-docker-1", "3e:50:6b:43:bb:32")

	if err := h.bootReconcile(ctx); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	entry, _ := h.state.Get("voip-asterisk-call-docker-1")
	if entry.IP != "172.24.0.109" {
		t.Errorf("Wrong match. expect: 172.24.0.109, got: %s", entry.IP)
	}
	if entry.AsteriskID != "" {
		t.Errorf("Wrong match. expect: a re-seeded entry to start unresolved, got: %s", entry.AsteriskID)
	}
}
