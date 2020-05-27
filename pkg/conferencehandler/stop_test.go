package conferencehandler

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestStop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &conferenceHandler{
		db:         mockDB,
		reqHandler: mockReq,
		cache:      mockCache,
	}

	type test struct {
		name            string
		conference      *conference.Conference
		expectBridgeIDs []string
	}

	tests := []test{
		{
			"type conference",
			&conference.Conference{
				ID:        uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
				Type:      conference.TypeConference,
				BridgeIDs: []string{},
			},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceSetStatus(gomock.Any(), tt.conference.ID, conference.StatusStopping)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			for _, bridgeID := range tt.expectBridgeIDs {
				mockDB.EXPECT().BridgeGet(gomock.Any(), bridgeID).Return(nil, fmt.Errorf("test"))
			}

			if err := h.Stop(tt.conference.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})

	}
}
