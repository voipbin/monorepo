package providercall

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	created := time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)
	updated := time.Date(2020, 4, 19, 3, 22, 17, 0, time.UTC)
	deleted := time.Date(2020, 4, 20, 3, 22, 17, 0, time.UTC)

	p := &ProviderCall{
		ID:           uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001"),
		CustomerID:   uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000001"),
		ProviderID:   uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000001"),
		FlowID:       uuid.FromStringOrNil("d4000000-0000-0000-0000-000000000001"),
		Source:       &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
		Destinations: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550000002"}},
		Anonymous:    "auto",
		CallIDs:      []uuid.UUID{uuid.FromStringOrNil("d5000000-0000-0000-0000-000000000001")},
		GroupcallIDs: []uuid.UUID{uuid.FromStringOrNil("d6000000-0000-0000-0000-000000000001")},
		TMCreate:     &created,
		TMUpdate:     &updated,
		TMDelete:     &deleted,
	}

	expect := &WebhookMessage{
		ID:           uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001"),
		CustomerID:   uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000001"),
		ProviderID:   uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000001"),
		FlowID:       uuid.FromStringOrNil("d4000000-0000-0000-0000-000000000001"),
		Source:       &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
		Destinations: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550000002"}},
		Anonymous:    "auto",
		CallIDs:      []uuid.UUID{uuid.FromStringOrNil("d5000000-0000-0000-0000-000000000001")},
		GroupcallIDs: []uuid.UUID{uuid.FromStringOrNil("d6000000-0000-0000-0000-000000000001")},
		TMCreate:     timePtr(time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)),
		TMUpdate:     timePtr(time.Date(2020, 4, 19, 3, 22, 17, 0, time.UTC)),
		TMDelete:     timePtr(time.Date(2020, 4, 20, 3, 22, 17, 0, time.UTC)),
	}

	res := p.ConvertWebhookMessage()
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	p := &ProviderCall{
		ID:           uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000002"),
		CustomerID:   uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000002"),
		ProviderID:   uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000002"),
		Anonymous:    "auto",
		CallIDs:      []uuid.UUID{uuid.FromStringOrNil("d5000000-0000-0000-0000-000000000002")},
		GroupcallIDs: []uuid.UUID{uuid.FromStringOrNil("d6000000-0000-0000-0000-000000000002")},
	}

	res, err := p.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := `{"id":"d1000000-0000-0000-0000-000000000002","customer_id":"d2000000-0000-0000-0000-000000000002","provider_id":"d3000000-0000-0000-0000-000000000002","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"anonymous":"auto","call_ids":["d5000000-0000-0000-0000-000000000002"],"groupcall_ids":["d6000000-0000-0000-0000-000000000002"],"tm_create":null,"tm_update":null,"tm_delete":null}`
	if string(res) != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, res)
	}
	if !json.Valid(res) {
		t.Errorf("Wrong match. expect: valid json")
	}
}

func timePtr(t time.Time) *time.Time { return &t }
