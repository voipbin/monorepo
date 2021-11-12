package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/requesthandler"
)

func TestTerminateCallNotExsist(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := conferenceHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
		cache:         mockCache,
	}

	tests := []struct {
		name       string
		conference *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("9f5001a6-9482-11eb-956e-f7ead445bb7a"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				ConfbridgeID: uuid.FromStringOrNil("4649cc0a-2086-11ec-8439-af4c561e87eb"),
				CallIDs:      []uuid.UUID{},
			},
		},
		{
			"have 1 call",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
				Type:   conference.TypeConference,
				Status: conference.StatusProgressing,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("2c4eaf4a-9482-11eb-9c2a-57de7ce9aed1"),
				},
			},
		},
		{
			"2 calls in the conference",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("fbf41954-0ab4-11eb-a22f-671a43bddb11"),
				Type:   conference.TypeConference,
				Status: conference.StatusProgressing,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("33a1af9a-9482-11eb-90d1-d7f2cf2288cb"),
					uuid.FromStringOrNil("6dfae364-9482-11eb-b11c-0f47944e2c54"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			mockDB.EXPECT().ConferenceSetStatus(gomock.Any(), tt.conference.ID, conference.StatusTerminating).Return(nil)

			for _, callID := range tt.conference.CallIDs {
				mockReq.EXPECT().CMCallsIDDelete(callID).Return(nil)
			}

			if len(tt.conference.CallIDs) == 0 {
				mockReq.EXPECT().CMConfbridgesIDDelete(tt.conference.ConfbridgeID).Return(nil)
				mockDB.EXPECT().ConferenceEnd(gomock.Any(), tt.conference.ID).Return(nil)
				mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
				mockNotify.EXPECT().NotifyEvent(notifyhandler.EventTypeConferenceDeleted, tt.conference.WebhookURI, tt.conference)
			}

			if err := h.Terminate(ctx, tt.conference.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})

	}
}
