package speaking

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"
)

// Speaking publishes on the global topic exchange (VOIP-1404/1419), so it must carry an explicit
// subscription address. The assertion pins the POINTER type: the event data reaches
// notifyhandler as a pointer and the interface assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Speaking)(nil)

func TestSpeakingEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	h := &Speaking{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ReferenceType: streaming.ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusActive,
	}

	res := h.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
	// A speaking session is an independent persistent record addressed by its own id, never by
	// the call or confbridge it is attached to.
	if res == referenceID.String() {
		t.Errorf("Speaking must not be addressed by its reference id. got: %s", res)
	}
	if res == customerID.String() {
		t.Errorf("Speaking must not be addressed by its customer id. got: %s", res)
	}
}

func Test_Status(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{
			name:   "StatusInitiating",
			status: StatusInitiating,
		},
		{
			name:   "StatusActive",
			status: StatusActive,
		},
		{
			name:   "StatusStopped",
			status: StatusStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status == "" {
				t.Errorf("expected non-empty status")
			}
		})
	}
}

func Test_Field(t *testing.T) {
	tests := []struct {
		name  string
		field Field
	}{
		{name: "FieldID", field: FieldID},
		{name: "FieldCustomerID", field: FieldCustomerID},
		{name: "FieldReferenceType", field: FieldReferenceType},
		{name: "FieldReferenceID", field: FieldReferenceID},
		{name: "FieldLanguage", field: FieldLanguage},
		{name: "FieldProvider", field: FieldProvider},
		{name: "FieldVoiceID", field: FieldVoiceID},
		{name: "FieldDirection", field: FieldDirection},
		{name: "FieldStatus", field: FieldStatus},
		{name: "FieldPodID", field: FieldPodID},
		{name: "FieldTMCreate", field: FieldTMCreate},
		{name: "FieldTMUpdate", field: FieldTMUpdate},
		{name: "FieldTMDelete", field: FieldTMDelete},
		{name: "FieldDeleted", field: FieldDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field == "" {
				t.Errorf("expected non-empty field")
			}
		})
	}
}
