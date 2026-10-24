package analysis

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func Test_CreateWebhookEvent(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	custID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	afID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		in   *Analysis

		expect map[string]any
	}{
		{
			name: "completed analysis drops the model",
			in: &Analysis{
				Identity:     commonidentity.Identity{ID: id, CustomerID: custID},
				ActiveflowID: afID,
				Status:       StatusCompleted,
				Result:       json.RawMessage(`{"overall_status":"ok"}`),
				Model:        "internal-model-name",
				Error:        "sanitized reason",
				TMCreate:     &created,
				TMUpdate:     &created,
			},
			expect: map[string]any{
				"id":            "11111111-1111-1111-1111-111111111111",
				"customer_id":   "22222222-2222-2222-2222-222222222222",
				"activeflow_id": "33333333-3333-3333-3333-333333333333",
				"status":        "completed",
				"result":        map[string]any{"overall_status": "ok"},
				"error":         "sanitized reason",
				"tm_create":     "2026-01-02T03:04:05Z",
				"tm_update":     "2026-01-02T03:04:05Z",
			},
		},
		{
			name: "empty result and error are omitted, absent times are null",
			in: &Analysis{
				Identity:     commonidentity.Identity{ID: id, CustomerID: custID},
				ActiveflowID: afID,
				Status:       StatusProgressing,
				Model:        "internal-model-name",
			},
			expect: map[string]any{
				"id":            "11111111-1111-1111-1111-111111111111",
				"customer_id":   "22222222-2222-2222-2222-222222222222",
				"activeflow_id": "33333333-3333-3333-3333-333333333333",
				"status":        "progressing",
				"tm_create":     nil,
				"tm_update":     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.in.CreateWebhookEvent()
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("Wrong match. could not decode the payload: %v", err)
			}
			if _, ok := got["model"]; ok {
				t.Errorf("Wrong match. the internal model must not be exposed")
			}
			if !reflect.DeepEqual(got, tt.expect) {
				keys := func(m map[string]any) []string {
					r := []string{}
					for k := range m {
						r = append(r, k)
					}
					sort.Strings(r)
					return r
				}
				t.Errorf("Wrong match.\nexpect keys: %v\ngot keys: %v\nexpect: %v\ngot: %v", keys(tt.expect), keys(got), tt.expect, got)
			}
		})
	}
}

func Test_ConvertWebhookMessage_fields(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	afID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	in := &Analysis{
		Identity:     commonidentity.Identity{ID: id},
		ActiveflowID: afID,
		Status:       StatusFailed,
		Result:       json.RawMessage(`{"a":1}`),
		Model:        "internal-model-name",
		Error:        "sanitized reason",
		TMCreate:     &created,
	}

	got := in.ConvertWebhookMessage()
	if got.ID != id || got.ActiveflowID != afID || got.Status != StatusFailed || got.Error != "sanitized reason" {
		t.Errorf("Wrong match. unexpected projection: %+v", got)
	}
	if string(got.Result) != `{"a":1}` {
		t.Errorf("Wrong match. unexpected result: %s", got.Result)
	}
	if got.TMCreate == nil || !got.TMCreate.Equal(created) || got.TMUpdate != nil {
		t.Errorf("Wrong match. unexpected times: %v %v", got.TMCreate, got.TMUpdate)
	}
}
