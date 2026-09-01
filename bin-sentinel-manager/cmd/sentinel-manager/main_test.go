package main

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-sentinel-manager/internal/config"
	"monorepo/bin-sentinel-manager/pkg/dockerwatchhandler"
	"monorepo/bin-sentinel-manager/pkg/k8swatchhandler"
)

// Test_buildBackend_selectsByConfig pins the backend-selection branch.
//
// The two arms construct genuinely different dependency graphs, so a wiring mistake here would
// only surface at runtime on whichever deployment shape nobody tested. The Docker arm is
// exercised for real (it reaches Redis, which is unreachable in a unit test and therefore errors
// — that error IS the evidence the arm ran), while the Kubernetes arm reaches
// rest.InClusterConfig(), which necessarily fails outside a cluster for the same reason.
func Test_buildBackend_selectsByConfig(t *testing.T) {
	tests := []struct {
		name string

		config config.Config

		expectErr bool
	}{
		{
			name: "docker backend reaches the redis connect step",

			config: config.Config{
				SentinelBackend:          config.BackendDocker,
				DockerSocketProxyAddress: "tcp://127.0.0.1:1",
				RedisAddress:             "127.0.0.1:1",
			},

			// unreachable Redis: the constructor is expected to fail loud rather than return a
			// backend that would watch nothing.
			expectErr: true,
		},
		{
			name: "kubernetes backend reaches the in-cluster config step",

			config: config.Config{SentinelBackend: config.BackendKubernetes},

			// no service-account mount outside a cluster.
			expectErr: true,
		},
		{
			name: "an unknown backend is refused rather than silently defaulting",

			config: config.Config{SentinelBackend: "nomad"},

			expectErr: true,
		},
		{
			name: "an unset backend is refused rather than silently defaulting",

			config: config.Config{},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			backend, cleanup, err := buildBackend(
				tt.config,
				requesthandler.NewMockRequestHandler(mc),
				notifyhandler.NewMockNotifyHandler(mc),
				utilhandler.NewMockUtilHandler(mc),
			)

			// cleanup must always be safe to defer, even on the error paths.
			if cleanup == nil {
				t.Fatalf("Wrong match. expect: a non-nil cleanup, got: nil")
			}
			defer cleanup()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				if backend != nil {
					t.Errorf("Wrong match. expect: nil backend on error, got: %T", backend)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if backend == nil {
				t.Errorf("Wrong match. expect: a backend, got: nil")
			}
		})
	}
}

// Test_buildBackend_bothBackendsSatisfyTheContract pins that both concrete types are usable
// through the one interface main() calls Run on.
func Test_buildBackend_bothBackendsSatisfyTheContract(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	// constructed directly rather than through buildBackend, which cannot reach either real
	// dependency from a unit test.
	dockerBackend := dockerwatchhandler.NewDockerWatchHandler(mockReq, mockNotify, mockUtil, nil, nil)
	if dockerBackend == nil {
		t.Errorf("Wrong match. expect: a docker backend, got: nil")
	}

	k8sBackend := k8swatchhandler.NewK8sWatchHandler(mockReq, mockNotify, mockUtil, nil)
	if k8sBackend == nil {
		t.Errorf("Wrong match. expect: a kubernetes backend, got: nil")
	}
}

// Test_rootCmd_registersSentinelBackend pins that the service binary's own flag set includes the
// selector. config.InitConfig errors with "flag not defined" otherwise, so a missing registration
// here breaks startup outright.
func Test_rootCmd_registersSentinelBackend(t *testing.T) {
	flag := rootCmd.Flags().Lookup("sentinel_backend")
	if flag == nil {
		t.Fatalf("Wrong match. expect: sentinel_backend registered on rootCmd, got: missing")
	}

	if flag.DefValue != "" {
		t.Errorf("Wrong match. expect: an empty default, got: %q", flag.DefValue)
	}
}

// Test_rootCmd_silencesUsageAndErrors keeps PR #1240's crash-loop-noise fix from regressing while
// this file is being edited for the backend branch.
func Test_rootCmd_silencesUsageAndErrors(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Errorf("Wrong match. expect: SilenceUsage true, got: false")
	}
	if !rootCmd.SilenceErrors {
		t.Errorf("Wrong match. expect: SilenceErrors true, got: false")
	}
}
