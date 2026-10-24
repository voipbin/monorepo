package activeflow

import (
	"encoding/json"
	"testing"
	"time"

	"monorepo/bin-webhook-manager/models/webhook"
)

func Test_IsPositive(t *testing.T) {
	tests := []struct {
		name string

		w *Webhook

		expectRes bool
	}{
		{
			name:      "nil entry",
			w:         nil,
			expectRes: false,
		},
		{
			name:      "positive entry",
			w:         &Webhook{URI: "https://example.com/hook", Method: webhook.MethodTypePOST},
			expectRes: true,
		},
		{
			name:      "tombstone with uri",
			w:         &Webhook{URI: "https://example.com/hook", Deleted: true},
			expectRes: false,
		},
		{
			name:      "empty uri",
			w:         &Webhook{Method: webhook.MethodTypePOST},
			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := tt.w.IsPositive(); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_MarshalJSON(t *testing.T) {
	tmDelete := time.Date(2026, 6, 10, 0, 0, 1, 0, time.UTC)

	tests := []struct {
		name string

		w *Webhook

		expectRes string
	}{
		{
			name: "positive entry encodes tm as unix nano",
			w: &Webhook{
				URI:    "https://example.com/hook",
				Method: webhook.MethodTypePOST,
				Tm:     time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
			},
			expectRes: `{"uri":"https://example.com/hook","method":"POST","tm":1781049600000000000}`,
		},
		{
			name: "tombstone keeps deleted flag and delete time",
			w: &Webhook{
				Deleted:  true,
				TMDelete: &tmDelete,
				Tm:       time.Unix(0, 1500),
			},
			expectRes: `{"deleted":true,"tm_delete":"2026-06-10T00:00:01Z","tm":1500}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := json.Marshal(tt.w)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if string(res) != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, string(res))
			}
		})
	}
}

func Test_MarshalJSON_invalid_delete_time(t *testing.T) {
	// a year outside [0,9999] cannot be encoded as RFC 3339.
	tmDelete := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	w := &Webhook{Deleted: true, TMDelete: &tmDelete}

	if _, err := json.Marshal(w); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string

		data string

		expectURI      string
		expectMethod   webhook.MethodType
		expectDeleted  bool
		expectTMDelete *time.Time
		expectTm       time.Time
	}{
		{
			name:          "positive entry",
			data:          `{"uri":"https://example.com/hook","method":"PUT","tm":1781049600000000000}`,
			expectURI:     "https://example.com/hook",
			expectMethod:  webhook.MethodTypePUT,
			expectDeleted: false,
			expectTm:      time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "tombstone",
			data:          `{"deleted":true,"tm_delete":"2026-06-10T00:00:01Z","tm":1500}`,
			expectDeleted: true,
			expectTMDelete: func() *time.Time {
				v := time.Date(2026, 6, 10, 0, 0, 1, 0, time.UTC)
				return &v
			}(),
			expectTm: time.Unix(0, 1500).UTC(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res Webhook
			if err := json.Unmarshal([]byte(tt.data), &res); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.URI != tt.expectURI || res.Method != tt.expectMethod || res.Deleted != tt.expectDeleted {
				t.Errorf("Wrong match. got: %+v", res)
			}
			if !res.Tm.Equal(tt.expectTm) || res.Tm.Location() != time.UTC {
				t.Errorf("Wrong match. expect tm: %v (UTC), got: %v (%v)", tt.expectTm, res.Tm, res.Tm.Location())
			}
			if (res.TMDelete == nil) != (tt.expectTMDelete == nil) {
				t.Fatalf("Wrong match. expect tm_delete: %v, got: %v", tt.expectTMDelete, res.TMDelete)
			}
			if tt.expectTMDelete != nil && !res.TMDelete.Equal(*tt.expectTMDelete) {
				t.Errorf("Wrong match. expect tm_delete: %v, got: %v", tt.expectTMDelete, res.TMDelete)
			}
		})
	}
}

func Test_UnmarshalJSON_invalid(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "not json", data: `{"uri":`},
		{name: "tm is not an integer", data: `{"uri":"x","tm":"2026-06-10T00:00:00Z"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res Webhook
			if err := json.Unmarshal([]byte(tt.data), &res); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}
