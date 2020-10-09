package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

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
			"empty option",
			&call.Call{},
			&action.Action{
				Type:   action.TypeConferenceJoin,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeConferenceJoin,
				ID:     uuid.Nil,
				Option: []byte(`{"conference_id":""}`),
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
			"empty option",
			&call.Call{},
			&action.Action{
				Type:   action.TypeStreamEcho,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeStreamEcho,
				ID:     uuid.Nil,
				Option: []byte(`{"duration":180000}`),
			},
			180 * 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelContinue(tt.call.AsteriskID, tt.call.ChannelID, "svc-stream_echo", "s", 1, "").Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, gomock.Any(), tt.expectAction).Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteAnswer(t *testing.T) {
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
			"empty option",
			&call.Call{
				ID:         uuid.FromStringOrNil("4371b0d6-df48-11ea-9a8c-177968c165e9"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "5b21353a-df48-11ea-8207-6fc0fa36a3fe",
			},
			&action.Action{
				Type:   action.TypeAnswer,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeAnswer,
				ID:     uuid.Nil,
				Option: []byte(`{}`),
			},
			180 * 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, 10, tt.expectAction).Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionTimeoutNext(t *testing.T) {
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
		call    *call.Call
		action  *action.Action
		channel *channel.Channel
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("66e039c6-e3fc-11ea-ae6f-53584373e7c9"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "12a05228-e3fd-11ea-b55f-afd68e7aa755",
				Action: action.Action{
					ID:        uuid.FromStringOrNil("b44bae7a-e3fc-11ea-a908-374a03455628"),
					TMExecute: "2020-04-18T03:22:17.995000",
				},
			},
			&action.Action{
				ID:        uuid.FromStringOrNil("b44bae7a-e3fc-11ea-a908-374a03455628"),
				Type:      action.TypeAnswer,
				Option:    []byte(`{}`),
				TMExecute: "2020-04-18T03:22:17.995000",
			},
			&channel.Channel{
				ID: "12a05228-e3fd-11ea-b55f-afd68e7aa755",
				Data: map[string]interface{}{
					"CONTEXT": "conf-in",
				},
				Stasis: "call-in",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.call.ChannelID).Return(tt.channel, nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)

			if err := h.ActionTimeout(tt.call.ID, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
