package metricshandler

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// Test_collectors_registered checks that every exported collector is registered
// with the default registry when the package is loaded: registering the same
// collector again must report that it already exists.
func Test_collectors_registered(t *testing.T) {
	tests := []struct {
		name      string
		collector prometheus.Collector
	}{
		{"receive request process time", ReceivedRequestProcessTime},
		{"receive subscribe event process time", ReceivedSubscribeEventProcessTime},
		{"db operation duration", DBOperationDuration},
		{"db operation total", DBOperationTotal},
		{"cache operation total", CacheOperationTotal},
		{"rpc call duration", RPCCallDuration},
		{"rpc call total", RPCCallTotal},
		{"login total", LoginTotal},
		{"password reset total", PasswordResetTotal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := prometheus.Register(tt.collector)

			var already prometheus.AlreadyRegisteredError
			if !errors.As(err, &already) {
				t.Errorf("Wrong match. expect the collector to be registered already, got: %v", err)
			}
		})
	}
}
