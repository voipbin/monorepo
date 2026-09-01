package dockerwatchhandler

import (
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockernetwork "github.com/docker/docker/api/types/network"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/cachehandler"
)

func Test_NewDockerWatchHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := NewDockerWatchHandler(
		requesthandler.NewMockRequestHandler(mc),
		notifyhandler.NewMockNotifyHandler(mc),
		utilhandler.NewMockUtilHandler(mc),
		NewMockdockerClient(mc),
		cachehandler.NewMockCacheHandler(mc),
	)

	if h == nil {
		t.Fatalf("Wrong match. expect: non-nil handler, got: nil")
	}

	res, ok := h.(*dockerWatchHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *dockerWatchHandler, got: %T", h)
	}
	if res.state == nil {
		t.Errorf("Wrong match. expect: initialized state table, got: nil")
	}
	if res.flap == nil {
		t.Errorf("Wrong match. expect: initialized flap tracker, got: nil")
	}
	if res.refreshInterval != refreshInterval {
		t.Errorf("Wrong match. expect: %v, got: %v", refreshInterval, res.refreshInterval)
	}
	if res.reconnectDelay != reconnectDelay {
		t.Errorf("Wrong match. expect: %v, got: %v", reconnectDelay, res.reconnectDelay)
	}

	// a zero healthyStreamLifetime silently DISABLES the longevity reset, which would let an idle
	// fleet's periodic proxy restarts accumulate into a self-inflicted exit. The constructor must
	// always set it.
	expectLifetime := time.Duration(healthyStreamLifetimeFactor) * reconnectDelay
	if res.healthyStreamLifetime != expectLifetime {
		t.Errorf("Wrong match. expect: %v, got: %v", expectLifetime, res.healthyStreamLifetime)
	}
	if res.healthyStreamLifetime <= res.reconnectDelay {
		t.Errorf("Wrong match. expect: the healthy lifetime to exceed one reconnect delay, got: %v vs %v", res.healthyStreamLifetime, res.reconnectDelay)
	}
}

// Test_matchWatchedContainer is the single most important guard in this file: the asterisk-proxy
// sidecars share their parent's name PREFIX, so a naive prefix match would watch them too and
// publish a spurious recovery trigger every time a sidecar restarts.
func Test_matchWatchedContainer(t *testing.T) {
	tests := []struct {
		name string

		containerName string

		expectService string
		expectOK      bool
	}{
		{
			name: "asterisk_call_replica_1",

			containerName: "voip-asterisk-call-docker-1",

			expectService: container.ServiceAsteriskCall,
			expectOK:      true,
		},
		{
			name: "asterisk_call_replica_2",

			containerName: "voip-asterisk-call-docker-2",

			expectService: container.ServiceAsteriskCall,
			expectOK:      true,
		},
		{
			name: "asterisk_conference",

			containerName: "voip-asterisk-conference-docker-1",

			expectService: container.ServiceAsteriskConference,
			expectOK:      true,
		},
		{
			name: "asterisk_registrar",

			containerName: "voip-asterisk-registrar-docker-2",

			expectService: container.ServiceAsteriskRegistrar,
			expectOK:      true,
		},
		{
			name: "docker_leading_slash_is_stripped",

			containerName: "/voip-asterisk-call-docker-1",

			expectService: container.ServiceAsteriskCall,
			expectOK:      true,
		},
		{
			name: "multi_digit_replica_index",

			containerName: "voip-asterisk-call-docker-12",

			expectService: container.ServiceAsteriskCall,
			expectOK:      true,
		},
		{
			name: "asterisk_call_proxy_sidecar_is_excluded",

			containerName: "voip-asterisk-call-docker-1-asterisk-call-proxy-1",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "asterisk_conference_proxy_sidecar_is_excluded",

			containerName: "voip-asterisk-conference-docker-2-asterisk-conference-proxy-1",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "prefix_without_replica_index_is_excluded",

			containerName: "voip-asterisk-call-docker-",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "unrelated_container",

			containerName: "voip-kamailio-docker-1",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "rtpengine_is_not_watched",

			containerName: "voip-rtpengine-docker-1",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "empty_name",

			containerName: "",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "name_is_a_superstring_of_the_prefix_but_not_a_replica",

			containerName: "voip-asterisk-call-docker-1a",

			expectService: "",
			expectOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resService, resOK := matchWatchedContainer(tt.containerName)

			if resOK != tt.expectOK {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectOK, resOK)
			}
			if resService != tt.expectService {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectService, resService)
			}
		})
	}
}

func Test_isReplicaIndex(t *testing.T) {
	tests := []struct {
		name string

		value string

		expect bool
	}{
		{name: "single_digit", value: "1", expect: true},
		{name: "multi_digit", value: "104", expect: true},
		{name: "zero", value: "0", expect: true},
		{name: "empty", value: "", expect: false},
		{name: "trailing_letter", value: "1a", expect: false},
		{name: "leading_letter", value: "a1", expect: false},
		{name: "dash", value: "1-2", expect: false},
		{name: "unicode_digit_is_rejected", value: "١", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isReplicaIndex(tt.value); res != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_containerNameOf(t *testing.T) {
	tests := []struct {
		name string

		summary dockercontainer.Summary

		expect string
	}{
		{
			name: "normal",

			summary: dockercontainer.Summary{Names: []string{"/voip-asterisk-call-docker-1"}},

			expect: "voip-asterisk-call-docker-1",
		},
		{
			name: "first_name_wins",

			summary: dockercontainer.Summary{Names: []string{"/voip-asterisk-call-docker-1", "/alias"}},

			expect: "voip-asterisk-call-docker-1",
		},
		{
			name: "no_names",

			summary: dockercontainer.Summary{},

			expect: "",
		},
		{
			name: "name_without_slash",

			summary: dockercontainer.Summary{Names: []string{"voip-asterisk-call-docker-1"}},

			expect: "voip-asterisk-call-docker-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := containerNameOf(tt.summary); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func Test_resolveContainerIP(t *testing.T) {
	tests := []struct {
		name string

		inspect dockercontainer.InspectResponse

		expect string
	}{
		{
			name: "production_network",

			inspect: dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*dockernetwork.EndpointSettings{
						"production": {IPAddress: "172.24.0.101"},
					},
				},
			},

			expect: "172.24.0.101",
		},
		{
			name: "production_wins_over_other_networks",

			inspect: dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*dockernetwork.EndpointSettings{
						"aaa-first-alphabetically": {IPAddress: "10.0.0.9"},
						"production":               {IPAddress: "172.24.0.101"},
					},
				},
			},

			expect: "172.24.0.101",
		},
		{
			name: "falls_back_to_another_network",

			inspect: dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*dockernetwork.EndpointSettings{
						"some-other-network": {IPAddress: "10.0.0.9"},
					},
				},
			},

			expect: "10.0.0.9",
		},
		{
			name: "fallback_is_deterministic_across_several_networks",

			inspect: dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*dockernetwork.EndpointSettings{
						"zzz-last":  {IPAddress: "10.0.0.9"},
						"aaa-first": {IPAddress: "10.0.0.2"},
					},
				},
			},

			expect: "10.0.0.2",
		},
		{
			name: "production_present_but_empty_falls_back",

			inspect: dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*dockernetwork.EndpointSettings{
						"production":         {IPAddress: ""},
						"some-other-network": {IPAddress: "10.0.0.9"},
					},
				},
			},

			expect: "10.0.0.9",
		},
		{
			name: "nil_endpoint_is_skipped",

			inspect: dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*dockernetwork.EndpointSettings{
						"production": nil,
						"other":      {IPAddress: "10.0.0.9"},
					},
				},
			},

			expect: "10.0.0.9",
		},
		{
			name: "nil_network_settings_dead_container",

			inspect: dockercontainer.InspectResponse{},

			expect: "",
		},
		{
			name: "no_addresses_anywhere_dead_container",

			inspect: dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*dockernetwork.EndpointSettings{
						"production": {IPAddress: ""},
					},
				},
			},

			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := resolveContainerIP(tt.inspect); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func Test_sortedNetworkNames(t *testing.T) {
	networks := map[string]*dockernetwork.EndpointSettings{
		"zulu":     {},
		"alpha":    {},
		"mike":     {},
		"bravo":    {},
		"november": {},
	}

	expect := []string{"alpha", "bravo", "mike", "november", "zulu"}

	// repeated because map iteration order varies per run; a non-deterministic implementation
	// would only fail intermittently on a single call.
	for i := 0; i < 20; i++ {
		res := sortedNetworkNames(networks)

		if len(res) != len(expect) {
			t.Fatalf("Wrong match. expect: %d entries, got: %d", len(expect), len(res))
		}
		for j := range expect {
			if res[j] != expect[j] {
				t.Fatalf("Wrong match. expect: %v, got: %v", expect, res)
			}
		}
	}
}

func Test_watchedContainerPrefixes(t *testing.T) {
	expect := map[string]string{
		"voip-asterisk-call-docker-":       "asterisk-call",
		"voip-asterisk-conference-docker-": "asterisk-conference",
		"voip-asterisk-registrar-docker-":  "asterisk-registrar",
	}

	if len(watchedContainerPrefixes) != len(expect) {
		t.Fatalf("Wrong match. expect: %d prefixes, got: %d", len(expect), len(watchedContainerPrefixes))
	}

	for prefix, service := range expect {
		res, ok := watchedContainerPrefixes[prefix]
		if !ok {
			t.Errorf("Wrong match. expect: prefix %s to be watched, got: missing", prefix)
			continue
		}
		if res != service {
			t.Errorf("Wrong match. expect: %s, got: %s", service, res)
		}
	}
}

func Test_prometheusMetrics(t *testing.T) {
	if metricsNamespace == "" {
		t.Errorf("Wrong match. expect: non-empty metrics namespace, got: empty")
	}
	if promContainerStateChangeCounter == nil {
		t.Errorf("Wrong match. expect: initialized promContainerStateChangeCounter, got: nil")
	}
	if promContainerUnresolvedAsteriskIDCounter == nil {
		t.Errorf("Wrong match. expect: initialized promContainerUnresolvedAsteriskIDCounter, got: nil")
	}
	if promContainerRefreshMissCounter == nil {
		t.Errorf("Wrong match. expect: initialized promContainerRefreshMissCounter, got: nil")
	}
}
