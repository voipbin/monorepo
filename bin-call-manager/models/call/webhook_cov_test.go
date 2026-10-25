package call

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func covTestCall() *Call {
	create := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	ringing := create.Add(time.Second)
	progressing := create.Add(2 * time.Second)
	hangup := create.Add(3 * time.Second)

	return &Call{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e01"),
			CustomerID: uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e02"),
		},
		FlowID:         uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e03"),
		ActiveflowID:   uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e04"),
		Type:           TypeFlow,
		MasterCallID:   uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e05"),
		ChainedCallIDs: []uuid.UUID{uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e06")},
		RecordingID:    uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e07"),
		RecordingIDs:   []uuid.UUID{uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e07")},
		Source:         commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"},
		Destination:    commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"},
		Status:         StatusRinging,
		Metadata:       map[string]interface{}{"key": "value"},
		Direction:      DirectionOutgoing,
		MuteDirection:  MuteDirectionBoth,
		HangupBy:       HangupByLocal,
		HangupReason:   HangupReasonNormal,
		TMRinging:      &ringing,
		TMProgressing:  &progressing,
		TMHangup:       &hangup,
		TMCreate:       &create,
	}
}

func Test_CallConvertWebhookMessage_cov(t *testing.T) {
	c := covTestCall()

	expect := &WebhookMessage{
		Identity:       c.Identity,
		Owner:          c.Owner,
		FlowID:         c.FlowID,
		ActiveflowID:   c.ActiveflowID,
		Type:           c.Type,
		MasterCallID:   c.MasterCallID,
		ChainedCallIDs: c.ChainedCallIDs,
		RecordingID:    c.RecordingID,
		RecordingIDs:   c.RecordingIDs,
		Source:         c.Source,
		Destination:    c.Destination,
		Status:         c.Status,
		Action:         c.Action,
		Metadata:       c.Metadata,
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

	if res := c.ConvertWebhookMessage(); !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_CallCreateWebhookEvent_cov(t *testing.T) {
	c := covTestCall()

	res, err := c.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the payload must be the JSON form of the converted webhook message
	var got WebhookMessage
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("could not unmarshal the event. err: %v", err)
	}
	if got.ID != c.ID || got.CustomerID != c.CustomerID || got.Status != StatusRinging || got.Direction != DirectionOutgoing {
		t.Errorf("Wrong match. got: %v", got)
	}

	var raw map[string]any
	if err := json.Unmarshal(res, &raw); err != nil {
		t.Fatalf("could not unmarshal the event. err: %v", err)
	}
	if raw["hangup_reason"] != string(HangupReasonNormal) || raw["status"] != string(StatusRinging) {
		t.Errorf("Wrong match. unexpected json keys: %v", raw)
	}
}
