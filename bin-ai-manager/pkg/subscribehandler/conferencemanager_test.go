package subscribehandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/pkg/summaryhandler"
	"monorepo/bin-common-handler/models/sock"
	cfconference "monorepo/bin-conference-manager/models/conference"
)

func TestProcessEventCMConferenceUpdated(t *testing.T) {
	tests := []struct {
		name      string
		event     *sock.Event
		setupMock func(*summaryhandler.MockSummaryHandler)
		wantError bool
	}{
		{
			// VOIP-1422: the event type here is conference_deleted, not
			// conference_updated -- see the topicPatterns doc comment in main.go and
			// processEventCMConferenceUpdated's own doc comment for why. This test
			// exercises the function directly (bypassing the processEvent switch), so
			// it never actually matched on m.Type either before or after that change;
			// updated to match production reality regardless.
			name: "processes_conference_deleted_event_successfully",
			event: func() *sock.Event {
				conf := &cfconference.Conference{
					Name:   "Test Conference",
					Status: cfconference.StatusTerminated,
				}
				conf.ID = uuid.Must(uuid.NewV4())
				conf.CustomerID = uuid.Must(uuid.NewV4())

				data, _ := json.Marshal(conf)
				return &sock.Event{
					Publisher: "conference-manager",
					Type:      string(cfconference.EventTypeConferenceDeleted),
					Data:      json.RawMessage(data),
				}
			}(),
			setupMock: func(m *summaryhandler.MockSummaryHandler) {
				m.EXPECT().EventCMConferenceUpdated(gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantError: false,
		},
		{
			name: "handles_invalid_json_data",
			event: &sock.Event{
				Publisher: "conference-manager",
				Type:      string(cfconference.EventTypeConferenceDeleted),
				Data:      json.RawMessage([]byte("invalid json")),
			},
			setupMock: func(m *summaryhandler.MockSummaryHandler) {
				// Should not be called on error
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSummaryHandler := summaryhandler.NewMockSummaryHandler(ctrl)
			tt.setupMock(mockSummaryHandler)

			h := &subscribeHandler{
				summaryHandler: mockSummaryHandler,
			}

			err := h.processEventCMConferenceUpdated(context.Background(), tt.event)
			if (err != nil) != tt.wantError {
				t.Errorf("processEventCMConferenceUpdated() error = %v, wantError %v", err, tt.wantError)
			}

			// Allow goroutine launched by processEventCMConferenceUpdated to complete
			time.Sleep(50 * time.Millisecond)
		})
	}
}
