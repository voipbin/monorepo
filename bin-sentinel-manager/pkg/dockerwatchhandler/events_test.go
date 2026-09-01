package dockerwatchhandler

import (
	"context"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerevents "github.com/docker/docker/api/types/events"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-sentinel-manager/models/asteriskaddress"
	"monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/cachehandler"
)

func containerEvent(action dockerevents.Action, id string, name string, timeNano int64) dockerevents.Message {
	return dockerevents.Message{
		Type:   dockerevents.ContainerEventType,
		Action: action,
		Actor: dockerevents.Actor{
			ID:         id,
			Attributes: map[string]string{"name": name},
		},
		Time:     timeNano / int64(time.Second),
		TimeNano: timeNano,
	}
}

func Test_watchedEventFilters(t *testing.T) {
	res := watchedEventFilters()

	if !res.ExactMatch("type", "container") {
		t.Errorf("Wrong match. expect: type=container filter, got: %v", res)
	}
	if !res.ExactMatch("event", "start") {
		t.Errorf("Wrong match. expect: event=start filter, got: %v", res)
	}
	if !res.ExactMatch("event", "die") {
		t.Errorf("Wrong match. expect: event=die filter, got: %v", res)
	}
	if res.ExactMatch("event", "destroy") {
		t.Errorf("Wrong match. expect: no event=destroy filter, got: present")
	}
}

// Test_sinceCursor pins the resume cursor. Docker's `since` is INCLUSIVE, so resuming at the last
// message's own timestamp would redeliver it -- the cursor must be one nanosecond past.
func Test_sinceCursor(t *testing.T) {
	tests := []struct {
		name string

		message dockerevents.Message

		expect string
	}{
		{
			name: "full_nanosecond_precision",

			message: dockerevents.Message{TimeNano: 1756713600123456789},

			expect: "1756713600.123456790",
		},
		{
			name: "advances_past_the_message",

			message: dockerevents.Message{TimeNano: 1756713600000000000},

			expect: "1756713600.000000001",
		},
		{
			name: "rolls_over_into_the_next_second",

			message: dockerevents.Message{TimeNano: 1756713600999999999},

			expect: "1756713601.000000000",
		},
		{
			name: "falls_back_to_second_precision",

			message: dockerevents.Message{Time: 1756713600},

			expect: "1756713600.000000001",
		},
		{
			name: "no_timestamp_at_all",

			message: dockerevents.Message{},

			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := sinceCursor(tt.message); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func Test_handleContainerStarted(t *testing.T) {
	tests := []struct {
		name string

		message dockerevents.Message
		inspect dockercontainer.InspectResponse

		expectIP string
	}{
		{
			name: "normal",

			message: containerEvent(dockerevents.ActionStart, "id-call-1", "voip-asterisk-call-docker-1", 1756713600000000000),
			inspect: inspectWithIP("172.24.0.101"),

			expectIP: "172.24.0.101",
		},
		{
			name: "no_resolvable_ip",

			message: containerEvent(dockerevents.ActionStart, "id-call-1", "voip-asterisk-call-docker-1", 1756713600000000000),
			inspect: dockercontainer.InspectResponse{},

			expectIP: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDocker := NewMockdockerClient(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &dockerWatchHandler{
				notifyHandler: mockNotify,
				dockerClient:  mockDocker,
				state:         newStateTable(),
				flap:          newFlapTracker(flapWindow, flapThreshold),
			}

			ctx := context.Background()
			mockDocker.EXPECT().ContainerInspect(ctx, "id-call-1").Return(tt.inspect, nil)
			mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerStarted, &container.Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       container.ServiceAsteriskCall,
				AsteriskID:    "",
			})

			if err := h.handleEvent(ctx, tt.message); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			entry, ok := h.state.Get("voip-asterisk-call-docker-1")
			if !ok {
				t.Fatalf("Wrong match. expect: entry created, got: missing")
			}
			if entry.IP != tt.expectIP {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectIP, entry.IP)
			}
			if entry.AsteriskID != "" {
				t.Errorf("Wrong match. expect: a new entry to start unresolved, got: %s", entry.AsteriskID)
			}
		})
	}
}

// Test_handleContainerStarted_inspectErrorStillCreatesEntry pins that an inspect failure does not
// cost the entry itself: the entry's EXISTENCE is what lets the eventual `die` publish at all.
func Test_handleContainerStarted_inspectErrorStillCreatesEntry(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &dockerWatchHandler{
		notifyHandler: mockNotify,
		dockerClient:  mockDocker,
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	ctx := context.Background()
	mockDocker.EXPECT().ContainerInspect(ctx, "id-call-1").Return(dockercontainer.InspectResponse{}, errors.New("no such container"))
	mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerStarted, gomock.Any())

	message := containerEvent(dockerevents.ActionStart, "id-call-1", "voip-asterisk-call-docker-1", 1756713600000000000)
	if err := h.handleEvent(ctx, message); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if _, ok := h.state.Get("voip-asterisk-call-docker-1"); !ok {
		t.Errorf("Wrong match. expect: entry created despite the inspect failure, got: missing")
	}
}

// Test_handleContainerDied is the core publish-path test: the id must come from the table, and
// NEVER from a die-time inspect or scan (the mocks would fail on any such call).
func Test_handleContainerDied(t *testing.T) {
	tests := []struct {
		name string

		entries []seededEntry
		message dockerevents.Message

		expectEvent *container.Event
	}{
		{
			name: "publishes_the_last_known_asterisk_id",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", "3e:50:6b:43:bb:32"},
			},
			message: containerEvent(dockerevents.ActionDie, "id-call-1", "voip-asterisk-call-docker-1", 1756713600000000000),

			expectEvent: &container.Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       container.ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},
		},
		{
			name: "publishes_an_unresolved_id_when_the_container_died_before_resolution",

			entries: []seededEntry{
				{"voip-asterisk-call-docker-1", container.ServiceAsteriskCall, "172.24.0.101", ""},
			},
			message: containerEvent(dockerevents.ActionDie, "id-call-1", "voip-asterisk-call-docker-1", 1756713600000000000),

			expectEvent: &container.Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       container.ServiceAsteriskCall,
				AsteriskID:    "",
			},
		},
		{
			name: "publishes_an_unresolved_id_when_there_is_no_table_entry_at_all",

			entries: nil,
			message: containerEvent(dockerevents.ActionDie, "id-conf-1", "voip-asterisk-conference-docker-1", 1756713600000000000),

			expectEvent: &container.Event{
				ContainerName: "voip-asterisk-conference-docker-1",
				Service:       container.ServiceAsteriskConference,
				AsteriskID:    "",
			},
		},
		{
			name: "registrar_service_is_carried_through",

			entries: []seededEntry{
				{"voip-asterisk-registrar-docker-2", container.ServiceAsteriskRegistrar, "172.24.0.122", "aa:bb:cc:dd:ee:ff"},
			},
			message: containerEvent(dockerevents.ActionDie, "id-reg-2", "voip-asterisk-registrar-docker-2", 1756713600000000000),

			expectEvent: &container.Event{
				ContainerName: "voip-asterisk-registrar-docker-2",
				Service:       container.ServiceAsteriskRegistrar,
				AsteriskID:    "aa:bb:cc:dd:ee:ff",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &dockerWatchHandler{
				notifyHandler: mockNotify,
				dockerClient:  NewMockdockerClient(mc),
				cacheHandler:  cachehandler.NewMockCacheHandler(mc),
				state:         newStateTable(),
				flap:          newFlapTracker(flapWindow, flapThreshold),
			}
			for _, e := range tt.entries {
				h.state.Create(e.containerName, e.service, e.ip, time.Now())
				if e.asteriskID != "" {
					h.state.Resolve(e.containerName, e.asteriskID)
				}
			}

			ctx := context.Background()
			mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerDied, tt.expectEvent)

			if err := h.handleEvent(ctx, tt.message); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// the entry is consumed by the death; a same-name replacement must rebuild it.
			if _, ok := h.state.Get(tt.expectEvent.ContainerName); ok {
				t.Errorf("Wrong match. expect: the entry removed on die, got: still present")
			}
		})
	}
}

// Test_handleEvent_startDieCycleRebuildsState walks the full lifecycle: start -> resolve -> die ->
// start again. The SECOND generation must not inherit the first's id.
func Test_handleEvent_startDieCycleRebuildsState(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &dockerWatchHandler{
		notifyHandler: mockNotify,
		dockerClient:  mockDocker,
		cacheHandler:  mockCache,
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	ctx := context.Background()
	name := "voip-asterisk-call-docker-1"

	// generation 1 starts and resolves.
	mockDocker.EXPECT().ContainerInspect(ctx, "id-gen-1").Return(inspectWithIP("172.24.0.101"), nil)
	mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerStarted, gomock.Any())
	if err := h.handleEvent(ctx, containerEvent(dockerevents.ActionStart, "id-gen-1", name, 1)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mockCache.EXPECT().AsteriskAddressInternalScan(ctx).Return([]*asteriskaddress.AsteriskAddress{
		{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
	}, nil)
	if err := h.refreshOnce(ctx); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// generation 1 dies, carrying its resolved id.
	mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerDied, &container.Event{
		ContainerName: name,
		Service:       container.ServiceAsteriskCall,
		AsteriskID:    "3e:50:6b:43:bb:32",
	})
	if err := h.handleEvent(ctx, containerEvent(dockerevents.ActionDie, "id-gen-1", name, 2)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// generation 2 reuses the container name and the static ip -- and must start clean.
	mockDocker.EXPECT().ContainerInspect(ctx, "id-gen-2").Return(inspectWithIP("172.24.0.101"), nil)
	mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerStarted, gomock.Any())
	if err := h.handleEvent(ctx, containerEvent(dockerevents.ActionStart, "id-gen-2", name, 3)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	entry, ok := h.state.Get(name)
	if !ok {
		t.Fatalf("Wrong match. expect: entry exists, got: missing")
	}
	if entry.AsteriskID != "" {
		t.Errorf("Wrong match. expect: generation 2 to start unresolved, got: %s", entry.AsteriskID)
	}
}

// Test_handleContainerDied_flapDamping pins design §3.4: past flapThreshold deaths inside
// flapWindow, further deaths are NOT published. The gomock expectation count is the assertion --
// a 4th publish would fail the controller.
func Test_handleContainerDied_flapDamping(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &dockerWatchHandler{
		notifyHandler: mockNotify,
		dockerClient:  NewMockdockerClient(mc),
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	ctx := context.Background()
	name := "voip-asterisk-call-docker-1"

	mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerDied, gomock.Any()).Times(flapThreshold)

	for i := 0; i < flapThreshold+3; i++ {
		if err := h.handleEvent(ctx, containerEvent(dockerevents.ActionDie, "id-call-1", name, int64(i+1))); err != nil {
			t.Fatalf("Wrong match at death %d. expect: ok, got: %v", i, err)
		}
	}
}

// Test_handleContainerDied_flapDampingIsPerContainer pins that a flapping container does not
// silence a healthy sibling.
func Test_handleContainerDied_flapDampingIsPerContainer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &dockerWatchHandler{
		notifyHandler: mockNotify,
		dockerClient:  NewMockdockerClient(mc),
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	ctx := context.Background()

	// flapThreshold publishes for the flapping one, plus one for the healthy sibling.
	mockNotify.EXPECT().PublishEvent(ctx, container.EventTypeContainerDied, gomock.Any()).Times(flapThreshold + 1)

	for i := 0; i < flapThreshold+3; i++ {
		if err := h.handleEvent(ctx, containerEvent(dockerevents.ActionDie, "id-1", "voip-asterisk-call-docker-1", int64(i+1))); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	if err := h.handleEvent(ctx, containerEvent(dockerevents.ActionDie, "id-2", "voip-asterisk-call-docker-2", 100)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_handleEvent_ignoredEvents pins that nothing outside the watched set ever publishes. The
// strict mock (no PublishEvent expectation at all) is the assertion.
func Test_handleEvent_ignoredEvents(t *testing.T) {
	tests := []struct {
		name string

		message dockerevents.Message
	}{
		{
			name: "unwatched_container_name",

			message: containerEvent(dockerevents.ActionDie, "id-x", "voip-kamailio-docker-1", 1),
		},
		{
			name: "asterisk_proxy_sidecar",

			message: containerEvent(dockerevents.ActionDie, "id-x", "voip-asterisk-call-docker-1-asterisk-call-proxy-1", 1),
		},
		{
			name: "missing_name_attribute",

			message: dockerevents.Message{
				Type:   dockerevents.ContainerEventType,
				Action: dockerevents.ActionDie,
				Actor:  dockerevents.Actor{ID: "id-x"},
			},
		},
		{
			name: "non_container_event_type",

			message: dockerevents.Message{
				Type:   dockerevents.NetworkEventType,
				Action: dockerevents.ActionDie,
				Actor:  dockerevents.Actor{ID: "id-x", Attributes: map[string]string{"name": "voip-asterisk-call-docker-1"}},
			},
		},
		{
			name: "unhandled_action",

			message: containerEvent(dockerevents.ActionDestroy, "id-x", "voip-asterisk-call-docker-1", 1),
		},
		{
			name: "health_status_action",

			message: containerEvent(dockerevents.ActionHealthStatusUnhealthy, "id-x", "voip-asterisk-call-docker-1", 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &dockerWatchHandler{
				notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
				dockerClient:  NewMockdockerClient(mc),
				state:         newStateTable(),
				flap:          newFlapTracker(flapWindow, flapThreshold),
			}

			if err := h.handleEvent(context.Background(), tt.message); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if h.state.Len() != 0 {
				t.Errorf("Wrong match. expect: no state entries, got: %d", h.state.Len())
			}
		})
	}
}

// Test_consumeEvents_advancesTheCursor pins the reconnect cursor: the stream must resume from the
// LAST delivered message, so a proxy restart does not silently drop a `die`.
func Test_consumeEvents_advancesTheCursor(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &dockerWatchHandler{
		notifyHandler: mockNotify,
		dockerClient:  mockDocker,
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	messages := make(chan dockerevents.Message, 2)
	errs := make(chan error, 1)

	messages <- containerEvent(dockerevents.ActionDie, "id-1", "voip-asterisk-call-docker-1", 1756713600000000000)
	// an event for an unwatched container still advances the cursor.
	messages <- containerEvent(dockerevents.ActionDie, "id-2", "voip-kamailio-docker-1", 1756713700000000000)
	close(messages)

	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Any()).
		Return((<-chan dockerevents.Message)(messages), (<-chan error)(errs))
	mockNotify.EXPECT().PublishEvent(gomock.Any(), container.EventTypeContainerDied, gomock.Any())

	res, delivered := h.consumeEvents(context.Background(), "")

	if res != "1756713700.000000001" {
		t.Errorf("Wrong match. expect: 1756713700.000000001, got: %s", res)
	}
	if !delivered {
		t.Errorf("Wrong match. expect: delivered=true, got: false")
	}
}

// Test_consumeEvents_returnsOnStreamError pins that a stream error ends the read so the caller can
// reconnect, and that the cursor survives the error.
func Test_consumeEvents_returnsOnStreamError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)

	h := &dockerWatchHandler{
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		dockerClient:  mockDocker,
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	messages := make(chan dockerevents.Message)
	errs := make(chan error, 1)
	errs <- errors.New("connection reset")

	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Any()).
		Return((<-chan dockerevents.Message)(messages), (<-chan error)(errs))

	res, delivered := h.consumeEvents(context.Background(), "1756713600.000000001")

	if res != "1756713600.000000001" {
		t.Errorf("Wrong match. expect: the cursor to survive the error, got: %s", res)
	}
	if delivered {
		t.Errorf("Wrong match. expect: delivered=false on a stream that produced only an error, got: true")
	}
}

// Test_consumeEvents_returnsOnContextCancel pins the shutdown path.
func Test_consumeEvents_returnsOnContextCancel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)

	h := &dockerWatchHandler{
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		dockerClient:  mockDocker,
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	messages := make(chan dockerevents.Message)
	errs := make(chan error)

	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Any()).
		Return((<-chan dockerevents.Message)(messages), (<-chan error)(errs))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan string, 1)
	go func() {
		res, _ := h.consumeEvents(ctx, "cursor")
		done <- res
	}()

	cancel()

	select {
	case res := <-done:
		if res != "cursor" {
			t.Errorf("Wrong match. expect: cursor, got: %s", res)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Wrong match. expect: consumeEvents to return on context cancel, got: still running")
	}
}

// Test_consumeEvents_passesTheSinceCursorToDocker pins that the resume cursor actually reaches the
// Events call rather than being computed and dropped.
func Test_consumeEvents_passesTheSinceCursorToDocker(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)

	h := &dockerWatchHandler{
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		dockerClient:  mockDocker,
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
	}

	messages := make(chan dockerevents.Message)
	close(messages)
	errs := make(chan error)

	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Cond(func(options dockerevents.ListOptions) bool {
			return options.Since == "1756713600.000000001"
		})).
		Return((<-chan dockerevents.Message)(messages), (<-chan error)(errs))

	h.consumeEvents(context.Background(), "1756713600.000000001")
}

// Test_Run_failsLoudOnBootError pins the top-level fail-loud contract: Run must return the boot
// error so cmd/sentinel-manager exits rather than running blind.
func Test_Run_failsLoudOnBootError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)

	h := &dockerWatchHandler{
		notifyHandler:   notifyhandler.NewMockNotifyHandler(mc),
		dockerClient:    mockDocker,
		cacheHandler:    cachehandler.NewMockCacheHandler(mc),
		state:           newStateTable(),
		flap:            newFlapTracker(flapWindow, flapThreshold),
		refreshInterval: time.Hour,
		reconnectDelay:  time.Hour,
	}

	mockDocker.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Return(nil, errors.New("connection refused"))

	if err := h.Run(context.Background()); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
}

// Test_Run_seedsThenRefreshesThenWatches pins the ORDER design §3.3 step 0 requires: seed, then
// ONE immediate refresh pass (not a wait-for-the-first-tick), then the event loop. A sentinel that
// started watching before seeding would treat its first observed death as unresolvable.
func Test_Run_seedsThenRefreshesThenWatches(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &dockerWatchHandler{
		notifyHandler: mockNotify,
		dockerClient:  mockDocker,
		cacheHandler:  mockCache,
		state:         newStateTable(),
		flap:          newFlapTracker(flapWindow, flapThreshold),
		// long enough that the background ticker never fires during this test: the resolution
		// below can only come from the IMMEDIATE pass.
		refreshInterval: time.Hour,
		reconnectDelay:  time.Hour,
	}

	ctx, cancel := context.WithCancel(context.Background())

	mockDocker.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Return([]dockercontainer.Summary{
		summaryOf("id-call-1", "voip-asterisk-call-docker-1"),
	}, nil)
	mockDocker.EXPECT().ContainerInspect(gomock.Any(), "id-call-1").Return(inspectWithIP("172.24.0.101"), nil)
	mockCache.EXPECT().AsteriskAddressInternalScan(gomock.Any()).Return([]*asteriskaddress.AsteriskAddress{
		{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
	}, nil).MinTimes(1)

	messages := make(chan dockerevents.Message)
	errs := make(chan error)
	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, dockerevents.ListOptions) (<-chan dockerevents.Message, <-chan error) {
			// the table must already be seeded AND resolved by the time watching starts.
			entry, ok := h.state.Get("voip-asterisk-call-docker-1")
			if !ok {
				t.Errorf("Wrong match. expect: the table seeded before watching, got: missing")
			} else if entry.AsteriskID != "3e:50:6b:43:bb:32" {
				t.Errorf("Wrong match. expect: the immediate refresh pass to have resolved the id, got: %q", entry.AsteriskID)
			}

			cancel()
			return messages, errs
		})

	if err := h.Run(ctx); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_runEventLoop_exitsAfterConsecutiveEmptyStreams pins the post-boot fail-loud path.
//
// Before this guard, a socket proxy that died AFTER boot left the loop retrying forever with only
// a log line -- sentinel "up" and watching nothing, exactly the failure mode design §3.2 calls
// worse than being visibly down. The loop must give up and return an error, which propagates to a
// non-zero process exit and a visible Komodo crash-loop.
func Test_runEventLoop_exitsAfterConsecutiveEmptyStreams(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)

	h := &dockerWatchHandler{
		notifyHandler:  notifyhandler.NewMockNotifyHandler(mc),
		dockerClient:   mockDocker,
		state:          newStateTable(),
		flap:           newFlapTracker(flapWindow, flapThreshold),
		reconnectDelay: time.Millisecond,
	}

	attempts := 0
	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, dockerevents.ListOptions) (<-chan dockerevents.Message, <-chan error) {
			attempts++
			// an immediately-closed message channel is what an unreachable proxy looks like:
			// the attempt ends without ever delivering an event.
			messages := make(chan dockerevents.Message)
			close(messages)
			return messages, make(chan error)
		}).
		Times(maxConsecutiveEmptyStreams)

	before := testutil.ToFloat64(promContainerEventStreamReconnectCounter.WithLabelValues(streamResultEmpty))

	err := h.runEventLoop(context.Background())
	if err == nil {
		t.Fatalf("Wrong match. expect: error after %d empty streams, got: nil", maxConsecutiveEmptyStreams)
	}

	if attempts != maxConsecutiveEmptyStreams {
		t.Errorf("Wrong match. expect: %d attempts, got: %d", maxConsecutiveEmptyStreams, attempts)
	}

	after := testutil.ToFloat64(promContainerEventStreamReconnectCounter.WithLabelValues(streamResultEmpty))
	if delta := after - before; delta != float64(maxConsecutiveEmptyStreams) {
		t.Errorf("Wrong match. expect: the empty-result counter to advance by %d, got: %v", maxConsecutiveEmptyStreams, delta)
	}
}

// Test_runEventLoop_deliveredStreamResetsTheFailureCount pins that the give-up counter tracks
// CONSECUTIVE failures. A stream that delivers events proves the proxy is reachable, so the budget
// must reset -- otherwise a long-lived sentinel would eventually exit from accumulated unrelated
// blips.
func Test_runEventLoop_deliveredStreamResetsTheFailureCount(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &dockerWatchHandler{
		notifyHandler:  mockNotify,
		dockerClient:   mockDocker,
		state:          newStateTable(),
		flap:           newFlapTracker(flapWindow, flapThreshold),
		reconnectDelay: time.Millisecond,
	}

	// the FIRST attempt delivers an event and then ends; every later attempt is empty. If the
	// counter reset works, the loop survives exactly one extra round.
	attempts := 0
	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, dockerevents.ListOptions) (<-chan dockerevents.Message, <-chan error) {
			attempts++

			messages := make(chan dockerevents.Message, 1)
			if attempts == 1 {
				messages <- containerEvent(dockerevents.ActionDie, "id-1", "voip-asterisk-call-docker-1", 1756713600000000000)
			}
			close(messages)

			return messages, make(chan error)
		}).
		Times(maxConsecutiveEmptyStreams + 1)

	mockNotify.EXPECT().PublishEvent(gomock.Any(), container.EventTypeContainerDied, gomock.Any())

	beforeDelivered := testutil.ToFloat64(promContainerEventStreamReconnectCounter.WithLabelValues(streamResultDelivered))

	if err := h.runEventLoop(context.Background()); err == nil {
		t.Fatalf("Wrong match. expect: error eventually, got: nil")
	}

	if attempts != maxConsecutiveEmptyStreams+1 {
		t.Errorf("Wrong match. expect: %d attempts (the delivering one resets the budget), got: %d", maxConsecutiveEmptyStreams+1, attempts)
	}

	afterDelivered := testutil.ToFloat64(promContainerEventStreamReconnectCounter.WithLabelValues(streamResultDelivered))
	if delta := afterDelivered - beforeDelivered; delta != 1 {
		t.Errorf("Wrong match. expect: the delivered-result counter to advance by 1, got: %v", delta)
	}
}

// Test_runEventLoop_stopsCleanlyOnContextCancel pins that a normal shutdown is NOT an error: a
// SIGTERM must exit 0, not crash-loop.
func Test_runEventLoop_stopsCleanlyOnContextCancel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDocker := NewMockdockerClient(mc)

	h := &dockerWatchHandler{
		notifyHandler:  notifyhandler.NewMockNotifyHandler(mc),
		dockerClient:   mockDocker,
		state:          newStateTable(),
		flap:           newFlapTracker(flapWindow, flapThreshold),
		reconnectDelay: time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())

	mockDocker.EXPECT().
		Events(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, dockerevents.ListOptions) (<-chan dockerevents.Message, <-chan error) {
			cancel()
			messages := make(chan dockerevents.Message)
			close(messages)
			return messages, make(chan error)
		})

	if err := h.runEventLoop(ctx); err != nil {
		t.Errorf("Wrong match. expect: nil on a clean shutdown, got: %v", err)
	}
}
