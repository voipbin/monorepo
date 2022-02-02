package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestHangup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotfiy,
	}

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}{
		{
			"normal",
			&channel.Channel{
				ID:          "70271162-1772-11ec-a941-fb10a2f9c2e7",
				AsteriskID:  "80:fa:5b:5e:da:81",
				HangupCause: ari.ChannelCauseNormalClearing,
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("7076de7c-1772-11ec-86f2-835e7382daf2"),
				ChannelID:  "70271162-1772-11ec-a941-fb10a2f9c2e7",
				AsteriskID: "80:fa:5b:5e:da:81",
				Status:     call.StatusProgressing,
				Action: action.Action{
					Type: action.TypeEcho,
				},
			},
		},
		{
			"chained calls",
			&channel.Channel{
				ID:          "e3c68930-1778-11ec-8c04-0bcef8a75b4f",
				AsteriskID:  "80:fa:5b:5e:da:81",
				HangupCause: ari.ChannelCauseNormalClearing,
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("e37dcd4e-1778-11ec-95c1-5b6f4657bd15"),
				ChannelID:  "e3c68930-1778-11ec-8c04-0bcef8a75b4f",
				AsteriskID: "80:fa:5b:5e:da:81",
				Status:     call.StatusProgressing,
				Action: action.Action{
					Type: action.TypeEcho,
				},
				ChainedCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("f8913c7a-1778-11ec-bcca-dbdc63ee1e38"),
					uuid.FromStringOrNil("f8e1cf1e-1778-11ec-ba6f-e73cb284ba93"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstBridgeDelete(gomock.Any(), tt.call.AsteriskID, tt.call.BridgeID).Return(nil)
			mockDB.EXPECT().CallSetHangup(gomock.Any(), tt.call.ID, call.HangupReasonNormal, call.HangupByRemote, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallHungup, gomock.Any())

			for _, chainedCallID := range tt.call.ChainedCallIDs {
				tmpCall := &call.Call{
					AsteriskID: "80:fa:5b:5e:da:81",
					ChannelID:  "b0c8ac74-1779-11ec-8038-fbb981f4ed27",
					ID:         chainedCallID,
					Status:     call.StatusProgressing,
				}

				mockDB.EXPECT().CallGet(gomock.Any(), chainedCallID).Return(tmpCall, nil)
				mockDB.EXPECT().CallSetStatus(gomock.Any(), tmpCall.ID, call.StatusTerminating, gomock.Any()).Return(nil)
				mockReq.EXPECT().AstChannelHangup(gomock.Any(), tmpCall.AsteriskID, tmpCall.ChannelID, ari.ChannelCauseNormalClearing)
			}
			if err := h.Hangup(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestHangupWithReason(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotfiy,
	}

	tests := []struct {
		name     string
		call     *call.Call
		reason   call.HangupReason
		hangupBy call.HangupBy
	}{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("7076de7c-1772-11ec-86f2-835e7382daf2"),
				ChannelID:  "70271162-1772-11ec-a941-fb10a2f9c2e7",
				AsteriskID: "80:fa:5b:5e:da:81",
				Status:     call.StatusProgressing,
				Action: action.Action{
					Type: action.TypeEcho,
				},
			},
			call.HangupReasonNormal,
			call.HangupByRemote,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetHangup(gomock.Any(), tt.call.ID, tt.reason, tt.hangupBy, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallHungup, tt.call)

			if err := h.HangupWithReason(context.Background(), tt.call, tt.reason, tt.hangupBy, getCurTime()); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestHanginUp(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotfiy,
	}

	tests := []struct {
		name  string
		call  *call.Call
		cause ari.ChannelCause
	}{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("785880aa-1777-11ec-abec-2b721201c1af"),
				ChannelID:  "7877dce8-1777-11ec-b4ea-3bb953ca2fe7",
				AsteriskID: "80:fa:5b:5e:da:81",
				Status:     call.StatusProgressing,
				Action: action.Action{
					Type: action.TypeEcho,
				},
			},
			ari.ChannelCauseNormalClearing,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, call.StatusTerminating, gomock.Any()).Return(nil)
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), tt.call.AsteriskID, tt.call.ChannelID, tt.cause).Return(nil)

			if err := h.HangingUp(context.Background(), tt.call.ID, tt.cause); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
