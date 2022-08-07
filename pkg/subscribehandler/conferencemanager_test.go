package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
)

func Test_processEventConferenceConferencecallJoined(t *testing.T) {
	tests := []struct {
		name string

		event *rabbitmqhandler.Event

		referenceID  uuid.UUID
		conferenceID uuid.UUID
	}{
		{
			"normal",

			&rabbitmqhandler.Event{
				Type:      cfconferencecall.EventTypeConferencecallJoined,
				Publisher: publisherConferenceManager,
				Data:      []byte(`{"reference_id":"318c5626-166b-11ed-b0a0-37590f049313", "conference_id":"378067d4-166b-11ed-a602-5744e189ee35"}`),
			},

			uuid.FromStringOrNil("318c5626-166b-11ed-b0a0-37590f049313"),
			uuid.FromStringOrNil("378067d4-166b-11ed-a602-5744e189ee35"),
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

			mockQueuecallHandler.EXPECT().Joined(gomock.Any(), tt.referenceID, tt.conferenceID)
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventConferenceConferencecallLeaved(t *testing.T) {
	tests := []struct {
		name string

		event *rabbitmqhandler.Event

		referenceID  uuid.UUID
		conferenceID uuid.UUID
	}{
		{
			"normal",

			&rabbitmqhandler.Event{
				Type:      cfconferencecall.EventTypeConferencecallLeaved,
				Publisher: publisherConferenceManager,
				Data:      []byte(`{"reference_id":"318c5626-166b-11ed-b0a0-37590f049313", "conference_id":"378067d4-166b-11ed-a602-5744e189ee35"}`),
			},

			uuid.FromStringOrNil("318c5626-166b-11ed-b0a0-37590f049313"),
			uuid.FromStringOrNil("378067d4-166b-11ed-a602-5744e189ee35"),
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

			mockQueuecallHandler.EXPECT().Leaved(gomock.Any(), tt.referenceID, tt.conferenceID)
			h.processEvent(tt.event)
		})
	}
}
