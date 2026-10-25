package outboundconfig

import (
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage_cov(t *testing.T) {
	create := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	update := create.Add(time.Hour)

	c := &OutboundConfig{
		ID:                            uuid.FromStringOrNil("8a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e01"),
		CustomerID:                    uuid.FromStringOrNil("8a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e02"),
		Name:                          "test name",
		Detail:                        "test detail",
		DestinationWhitelist:          []string{"us", "kr"},
		Codecs:                        "ulaw,alaw",
		DefaultOutgoingSourceNumberID: uuid.FromStringOrNil("8a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e03"),
		TMCreate:                      &create,
		TMUpdate:                      &update,
	}

	expect := &WebhookMessage{
		ID:                            c.ID,
		CustomerID:                    c.CustomerID,
		Name:                          "test name",
		Detail:                        "test detail",
		DestinationWhitelist:          []string{"us", "kr"},
		Codecs:                        "ulaw,alaw",
		DefaultOutgoingSourceNumberID: c.DefaultOutgoingSourceNumberID,
		TMCreate:                      &create,
		TMUpdate:                      &update,
		TMDelete:                      nil,
	}

	if res := ConvertWebhookMessage(c); !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_ConvertWebhookMessage_nil_cov(t *testing.T) {
	if res := ConvertWebhookMessage(nil); res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}
