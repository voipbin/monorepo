package callhandler

import (
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestARIChannelDestroyedContextTypeCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
		{
			"call normal destroy",
			&channel.Channel{
				ID: "31384bbc-dd97-11ea-9e42-433e5113c783",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
				},
				HangupCause: ari.ChannelCauseNormalClearing,
			},
			&call.Call{
				ChannelID: "31384bbc-dd97-11ea-9e42-433e5113c783",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf("no call"))

			if err := h.ARIChannelDestroyed(tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelDestroyedContextTypeConference(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
	}

	tests := []test{
		{
			"conference normal destroy",
			&channel.Channel{
				ID: "78ff0ed4-dd7b-11ea-9add-dbca62f7e8b9",
				Data: map[string]interface{}{
					"CONTEXT": "conf-in",
				},
				HangupCause: ari.ChannelCauseNormalClearing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := h.ARIChannelDestroyed(tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
