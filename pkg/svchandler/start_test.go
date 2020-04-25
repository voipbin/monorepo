package svchandler

import (
	"testing"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

func TestGetService(t *testing.T) {
	type test struct {
		name          string
		channel       *channel.Channel
		expectService service
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				Data: map[string]interface{}{
					"CONTEXT": contextIncomingVoip,
					"DOMAIN":  domainEcho,
				},
			},
			svcEcho,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := getService(tt.channel)
			if service != tt.expectService {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectService, service)
			}
		})
	}
}
