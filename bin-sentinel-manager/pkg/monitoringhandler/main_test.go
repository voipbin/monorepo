package monitoringhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"go.uber.org/mock/gomock"
)

func TestNewMonitoringHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates_new_monitoring_handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := NewMonitoringHandler(mockReq, mockNotify, mockUtil)

			if h == nil {
				t.Errorf("Expected non-nil handler, got nil")
			}
		})
	}
}

func TestNewMonitoringHandlerWithNilDependencies(t *testing.T) {
	tests := []struct {
		name string

		reqHandler    requesthandler.RequestHandler
		notifyHandler notifyhandler.NotifyHandler
		utilHandler   utilhandler.UtilHandler
	}{
		{
			name: "creates_handler_with_nil_request_handler",

			reqHandler:    nil,
			notifyHandler: notifyhandler.NewMockNotifyHandler(gomock.NewController(t)),
			utilHandler:   utilhandler.NewMockUtilHandler(gomock.NewController(t)),
		},
		{
			name: "creates_handler_with_nil_notify_handler",

			reqHandler:    requesthandler.NewMockRequestHandler(gomock.NewController(t)),
			notifyHandler: nil,
			utilHandler:   utilhandler.NewMockUtilHandler(gomock.NewController(t)),
		},
		{
			name: "creates_handler_with_nil_util_handler",

			reqHandler:    requesthandler.NewMockRequestHandler(gomock.NewController(t)),
			notifyHandler: notifyhandler.NewMockNotifyHandler(gomock.NewController(t)),
			utilHandler:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that NewMonitoringHandler doesn't panic with nil dependencies
			// In production, nil dependencies would cause issues, but the constructor should not panic
			h := NewMonitoringHandler(tt.reqHandler, tt.notifyHandler, tt.utilHandler)

			if h == nil {
				t.Errorf("Expected non-nil handler even with nil dependencies, got nil")
			}
		})
	}
}

func TestConstants(t *testing.T) {
	tests := []struct {
		name string

		constant      string
		expectedValue string
	}{
		{
			name: "namespace_voip_constant",

			constant:      namespaceVOIP,
			expectedValue: "voip",
		},
		{
			name: "namespace_bin_constant",

			constant:      namespaceBIN,
			expectedValue: "bin",
		},
		{
			name: "label_app_asterisk_call_constant",

			constant:      lableAppAsteriskCall,
			expectedValue: "asterisk-call",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expectedValue {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expectedValue, tt.constant)
			}
		})
	}
}

func TestPrometheusMetrics(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "prometheus_metrics_are_initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify metrics are not nil (they should be initialized in init())
			if promPodStateChangeCounter == nil {
				t.Errorf("Expected promPodStateChangeCounter to be initialized, got nil")
			}

			if metricsNamespace == "" {
				t.Errorf("Expected metricsNamespace to be non-empty, got empty string")
			}
		})
	}
}
