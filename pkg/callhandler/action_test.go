package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestActionExecuteEchoLegacy(t *testing.T) {
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
		name          string
		call          *call.Call
		action        *action.Action
		expectAction  *action.Action
		expectTimeout int
		expectReqConf *conference.Conference
	}

	tests := []test{
		{
			"empty action",
			&call.Call{},
			&action.Action{
				Type:   action.TypeEchoLegacy,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeEchoLegacy,
				ID:     uuid.Nil,
				Option: []byte(`{"duration":180000,"dtmf":false}`),
			},
			180 * 1000,
			&conference.Conference{
				Type:    conference.TypeEcho,
				Name:    "echo",
				Detail:  "action echo",
				Timeout: 180,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockConf.EXPECT().Start(tt.expectReqConf, tt.call).Return(nil, nil)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, tt.expectTimeout, tt.expectAction)

			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteConferenceJoin(t *testing.T) {
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
		name              string
		call              *call.Call
		action            *action.Action
		expectAction      *action.Action
		expectConfereneID uuid.UUID
	}

	tests := []test{
		{
			"empty action",
			&call.Call{},
			&action.Action{
				Type:   action.TypeConferenceJoin,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeConferenceJoin,
				ID:     uuid.Nil,
				Option: []byte(`{"conference_id":""}`),
				// Option: []byte(`{"duration":180000,"dtmf":false}`),
			},
			uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockConf.EXPECT().Join(tt.expectConfereneID, tt.call.ID)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteStreamEcho(t *testing.T) {
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
		name          string
		call          *call.Call
		action        *action.Action
		expectAction  *action.Action
		expectTimeout int
	}

	tests := []test{
		{
			"empty action",
			&call.Call{},
			&action.Action{
				Type:   action.TypeStreamEcho,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeStreamEcho,
				ID:     uuid.Nil,
				Option: []byte(`{}`),
			},
			180 * 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelContinue(tt.call.AsteriskID, tt.call.ChannelID, "svc-stream_echo", "s", 1, "").Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
