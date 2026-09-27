package hook

import (
	"testing"
)

func TestHookStruct(t *testing.T) {
	data := []byte(`{"key": "value"}`)

	h := Hook{
		ReceivedURI:  "/v1/webhooks/telnyx",
		ReceivedData: data,
	}

	if h.ReceivedURI != "/v1/webhooks/telnyx" {
		t.Errorf("Hook.ReceivedURI = %v, expected %v", h.ReceivedURI, "/v1/webhooks/telnyx")
	}
	if string(h.ReceivedData) != `{"key": "value"}` {
		t.Errorf("Hook.ReceivedData = %v, expected %v", string(h.ReceivedData), `{"key": "value"}`)
	}
}

func TestHookWithEmptyData(t *testing.T) {
	h := Hook{
		ReceivedURI:  "/v1/webhooks/messagebird",
		ReceivedData: nil,
	}

	if h.ReceivedURI != "/v1/webhooks/messagebird" {
		t.Errorf("Hook.ReceivedURI = %v, expected %v", h.ReceivedURI, "/v1/webhooks/messagebird")
	}
	if h.ReceivedData != nil {
		t.Errorf("Hook.ReceivedData should be nil, got %v", h.ReceivedData)
	}
}

func TestHookWithDifferentURIs(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"telnyx_webhook", "/v1/webhooks/telnyx"},
		{"messagebird_webhook", "/v1/webhooks/messagebird"},
		{"twilio_webhook", "/v1/webhooks/twilio"},
		{"email_webhook", "/v1/emails"},
		{"message_webhook", "/v1/messages"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Hook{
				ReceivedURI: tt.uri,
			}
			if h.ReceivedURI != tt.uri {
				t.Errorf("Hook.ReceivedURI = %v, expected %v", h.ReceivedURI, tt.uri)
			}
		})
	}
}
