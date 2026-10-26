package call

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"
)

func newTestWebhookCall(groupcallID uuid.UUID) *Call {
	tm := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	return &Call{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		},
		FlowID:         uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
		ActiveflowID:   uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
		Type:           TypeFlow,
		MasterCallID:   uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
		ChainedCallIDs: []uuid.UUID{uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")},
		RecordingID:    uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888"),
		RecordingIDs:   []uuid.UUID{uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")},
		GroupcallID:    groupcallID,
		Source:         commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"},
		Destination:    commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"},
		Status:         StatusProgressing,
		Action:         fmaction.Action{Type: fmaction.TypeAnswer},
		Direction:      DirectionOutgoing,
		MuteDirection:  MuteDirectionNone,
		HangupBy:       HangupByRemote,
		HangupReason:   HangupReasonNormal,
		TMRinging:      &tm,
		TMProgressing:  &tm,
		TMHangup:       &tm,
		TMCreate:       &tm,
		TMUpdate:       &tm,
		TMDelete:       &tm,
	}
}

func Test_ConvertWebhookMessage_groupcallID(t *testing.T) {
	groupcallID := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")

	tests := []struct {
		name        string
		groupcallID uuid.UUID
	}{
		{
			name:        "call is a groupcall leg",
			groupcallID: groupcallID,
		},
		{
			name:        "call is not part of a groupcall",
			groupcallID: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestWebhookCall(tt.groupcallID)

			expected := &WebhookMessage{
				Identity:       c.Identity,
				Owner:          c.Owner,
				FlowID:         c.FlowID,
				ActiveflowID:   c.ActiveflowID,
				Type:           c.Type,
				MasterCallID:   c.MasterCallID,
				ChainedCallIDs: c.ChainedCallIDs,
				RecordingID:    c.RecordingID,
				RecordingIDs:   c.RecordingIDs,
				GroupcallID:    tt.groupcallID,
				Source:         c.Source,
				Destination:    c.Destination,
				Status:         c.Status,
				Action:         c.Action,
				Direction:      c.Direction,
				MuteDirection:  c.MuteDirection,
				HangupBy:       c.HangupBy,
				HangupReason:   c.HangupReason,
				TMRinging:      c.TMRinging,
				TMProgressing:  c.TMProgressing,
				TMHangup:       c.TMHangup,
				TMCreate:       c.TMCreate,
				TMUpdate:       c.TMUpdate,
				TMDelete:       c.TMDelete,
			}

			res := c.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, expected) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expected, res)
			}
		})
	}
}

func Test_CreateWebhookEvent_groupcallID(t *testing.T) {
	tests := []struct {
		name        string
		groupcallID uuid.UUID
		expect      string
	}{
		{
			name:        "call is a groupcall leg",
			groupcallID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"),
			expect:      "99999999-9999-9999-9999-999999999999",
		},
		{
			name:        "call is not part of a groupcall",
			groupcallID: uuid.Nil,
			expect:      "00000000-0000-0000-0000-000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestWebhookCall(tt.groupcallID)

			data, err := c.CreateWebhookEvent()
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			res := map[string]interface{}{}
			if errUnmarshal := json.Unmarshal(data, &res); errUnmarshal != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errUnmarshal)
				return
			}

			if res["groupcall_id"] != tt.expect {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, res["groupcall_id"])
			}
		})
	}
}
