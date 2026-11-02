package contactstar

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	created := time.Date(2026, 11, 2, 9, 15, 30, 123456000, time.UTC)
	star := &ContactStar{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("6f1c1a52-8d3b-4c1e-9a5e-0b7f2d9a1c10"),
			CustomerID: uuid.FromStringOrNil("11111111-8d3b-4c1e-9a5e-0b7f2d9a1c10"),
		},
		AgentID:   uuid.FromStringOrNil("22222222-8d3b-4c1e-9a5e-0b7f2d9a1c10"),
		ContactID: uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901"),
		TMCreate:  &created,
	}

	res := star.ConvertWebhookMessage()
	if res.ContactID != star.ContactID || !res.Starred || res.TMCreate == nil || !res.TMCreate.Equal(created) {
		t.Errorf("Wrong match. got: %+v", res)
	}

	// the internal ids are never exposed
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	for _, leaked := range []string{star.ID.String(), star.CustomerID.String(), star.AgentID.String()} {
		if strings.Contains(string(data), leaked) {
			t.Errorf("Wrong match. internal id leaked: %s", data)
		}
	}
}

func Test_WebhookMessage_Unstarred(t *testing.T) {
	res := &WebhookMessage{ContactID: uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901")}

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if strings.Contains(string(data), "tm_create") || !strings.Contains(string(data), `"starred":false`) {
		t.Errorf("Wrong match. got: %s", data)
	}
}
