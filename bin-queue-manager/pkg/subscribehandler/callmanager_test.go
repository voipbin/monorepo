package subscribehandler

import (
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-queue-manager/pkg/queuecallhandler"
)

func Test_processEventCMConfbridgeJoined(t *testing.T) {
	tests := []struct {
		name string

		event *rabbitmqhandler.Event

		callID       uuid.UUID
		confbridgeID uuid.UUID
	}{
		{
			"normal",

			&rabbitmqhandler.Event{
				Type:      cmconfbridge.EventTypeConfbridgeJoined,
				Publisher: publisherCallManager,
				Data:      []byte(`{"id":"318c5626-166b-11ed-b0a0-37590f049313", "joined_call_id":"378067d4-166b-11ed-a602-5744e189ee35"}`),
			},

			uuid.FromStringOrNil("378067d4-166b-11ed-a602-5744e189ee35"),
			uuid.FromStringOrNil("318c5626-166b-11ed-b0a0-37590f049313"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockQueuecallHandler := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &subscribeHandler{
				queuecallHandler: mockQueuecallHandler,
			}

			mockQueuecallHandler.EXPECT().EventCallConfbridgeJoined(gomock.Any(), tt.callID, tt.confbridgeID)
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventCMConfbridgeLeaved(t *testing.T) {
	tests := []struct {
		name string

		event *rabbitmqhandler.Event

		callID       uuid.UUID
		confbridgeID uuid.UUID
	}{
		{
			"normal",

			&rabbitmqhandler.Event{
				Type:      cmconfbridge.EventTypeConfbridgeLeaved,
				Publisher: publisherCallManager,
				Data:      []byte(`{"id":"e2f30ff0-61e6-4922-8ec5-5e6ef2b3510b", "leaved_call_id":"b8d04427-972a-446b-8f03-0ff1ff77673e"}`),
			},

			uuid.FromStringOrNil("b8d04427-972a-446b-8f03-0ff1ff77673e"),
			uuid.FromStringOrNil("e2f30ff0-61e6-4922-8ec5-5e6ef2b3510b"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockQueuecallHandler := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &subscribeHandler{
				queuecallHandler: mockQueuecallHandler,
			}

			mockQueuecallHandler.EXPECT().EventCallConfbridgeLeaved(gomock.Any(), tt.callID, tt.confbridgeID)
			h.processEvent(tt.event)
		})
	}
}
