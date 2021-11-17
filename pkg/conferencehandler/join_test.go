package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/notifyhandler"
)

func TestJoin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := conferenceHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		cache:         mockCache,
		notifyHandler: mockNotify,
	}

	type test struct {
		name       string
		conference *conference.Conference
		callID     uuid.UUID
	}

	tests := []test{
		{
			"normal",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("89856980-9f1c-11ea-a2e8-272863862e18"),
				Type:         conference.TypeConference,
				ConfbridgeID: uuid.FromStringOrNil("7d0bb11c-3e69-11ec-a38a-7b47fb83fb56"),
			},
			uuid.FromStringOrNil("2f553862-3e69-11ec-84d8-f39902bb6f1e"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			// mockReq.EXPECT().CMConfbridgesIDCallsIDPost(tt.conference.ConfbridgeID, tt.callID).Return(nil)
			mockReq.EXPECT().CMV1ConfbridgeCallAdd(gomock.Any(), tt.conference.ConfbridgeID, tt.callID).Return(nil)

			ctx := context.Background()
			if err := h.Join(ctx, tt.conference.ID, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
