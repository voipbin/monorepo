package subscribehandler

import (
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	call "monorepo/bin-call-manager/models/call"

	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// Test_flushBatch_MixedBatch verifies that a batch mixing a peer_events-
// eligible event (call_hangup) with an ineligible-publisher event
// (registrar-manager, unrelated to peer_events) inserts BOTH into `events`
// (unaffected row count) AND exactly one row into `peer_events` (only the
// eligible entry). This is the integration-level check from the design
// doc's test plan §9.
func Test_flushBatch_MixedBatch(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	customerID := uuid.Must(uuid.NewV4())
	callID := uuid.Must(uuid.NewV4())
	msg := call.WebhookMessage{
		Identity:    commonidentity.Identity{ID: callID, CustomerID: customerID},
		Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"},
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"},
		Direction:   call.DirectionIncoming,
	}

	entries := []eventEntry{
		{
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      call.EventTypeCallHangup,
				Data:      mustJSON(t, msg),
			},
			receivedAt: time.Now(),
		},
		{
			event: &sock.Event{
				// Not eligible for peer_events (registrar-manager events
				// carry no Source/Destination/Self/Peer shape); this row
				// still counts toward the `events` batch.
				Publisher: "registrar-manager",
				Type:      "registrar_created",
				Data:      []byte(`{}`),
			},
			receivedAt: time.Now(),
		},
	}

	// events: both entries always land here, unconditionally.
	mockDB.EXPECT().EventBatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ interface{}, rows []dbhandler.EventRow) error {
			if len(rows) != 2 {
				t.Errorf("EventBatchInsert got %d rows, want 2 (both entries, unaffected by peer_events eligibility)", len(rows))
			}
			return nil
		},
	)

	// peer_events: only the eligible call_hangup entry.
	mockDB.EXPECT().PeerEventBatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ interface{}, rows []dbhandler.PeerEventRow) error {
			if len(rows) != 1 {
				t.Fatalf("PeerEventBatchInsert got %d rows, want 1 (only the eligible call_hangup entry)", len(rows))
			}
			if rows[0].ReferenceID != callID {
				t.Errorf("PeerEventBatchInsert row ReferenceID = %v, want %v", rows[0].ReferenceID, callID)
			}
			return nil
		},
	)

	h := &subscribeHandler{dbHandler: mockDB}
	h.flushBatch(entries)
}
