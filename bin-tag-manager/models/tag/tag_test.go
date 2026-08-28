package tag

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Tag carries an explicit subscription address on the global topic exchange
// (VOIP-1404/1419). The assertion pins the POINTER type: the event data reaches
// notifyhandler as a pointer and the interface assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Tag)(nil)

// TestTagEventSubscriptionID pins the address choice: a tag event is addressed by the
// tag's OWN id, never the customer it belongs to. Both fields are set to distinct
// UUIDs so returning the wrong one fails the test.
func TestTagEventSubscriptionID(t *testing.T) {
	tagID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	data := &Tag{
		Identity: commonidentity.Identity{
			ID:         tagID,
			CustomerID: customerID,
		},
		Name:   "vip",
		Detail: "vip customers",
	}

	res := data.EventSubscriptionID()
	if res != tagID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", tagID.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Tag must not be addressed by its customer id. got: %s", res)
	}
}

func TestTag(t *testing.T) {
	tests := []struct {
		name string

		tagName string
		detail  string
	}{
		{
			name: "creates_tag_with_all_fields",

			tagName: "VIP Customer",
			detail:  "High value customer tag",
		},
		{
			name: "creates_tag_with_empty_fields",

			tagName: "",
			detail:  "",
		},
		{
			name: "creates_tag_with_special_characters",

			tagName: "Customer-Tag_123",
			detail:  "Tag with special chars: !@#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := &Tag{
				Name:   tt.tagName,
				Detail: tt.detail,
			}

			if tag.Name != tt.tagName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.tagName, tag.Name)
			}
			if tag.Detail != tt.detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.detail, tag.Detail)
			}
		})
	}
}
