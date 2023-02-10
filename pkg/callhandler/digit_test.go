package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_digitsReceivedNotActionDTMFReceived(t *testing.T) {

	tests := []struct {
		name     string
		channel  *channel.Channel
		call     *call.Call
		digit    string
		duration int

		expectVariables map[string]string
	}{
		{
			"normal",
			&channel.Channel{
				ID:         "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("b2a45cf6-9ace-11ea-9354-4baa7f3ad331"),
				ChannelID: "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				Action: fmaction.Action{
					Type: fmaction.TypeEcho,
				},
			},
			"4",
			100,

			map[string]string{
				variableCallDigits: "4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.call.ActiveFlowID, tt.expectVariables).Return(nil)

			if err := h.digitsReceived(ctx, tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_DTMFReceived_action_digits_receive_continue(t *testing.T) {

	tests := []struct {
		name string

		channel  *channel.Channel
		digit    string
		duration int

		responseCall *call.Call
		responseVar  *variable.Variable
		savedDTMFs   string

		expectVariables map[string]string
	}{
		{
			"length not enough",
			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			"4",
			100,

			&call.Call{
				ID:        uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type:   fmaction.TypeDigitsReceive,
					Option: []byte(`{"length": 3}`),
				},
			},
			&variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "",
				},
			},
			"",

			map[string]string{
				variableCallDigits: "${" + variableCallDigits + "}4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.responseCall, nil)
			mockReq.EXPECT().FlowV1VariableGet(gomock.Any(), tt.responseCall.ActiveFlowID).Return(tt.responseVar, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.responseCall.ActiveFlowID, tt.expectVariables).Return(nil)

			if err := h.digitsReceived(ctx, tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Test_DTMFReceived_action_digits_receive_stop(t *testing.T) {

	tests := []struct {
		name string

		channel  *channel.Channel
		digit    string
		duration int

		expectDigits string

		responseCall     *call.Call
		responseVariable *variable.Variable

		expectVariables map[string]string
	}{
		{
			"finish key #",

			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			"#",
			100,

			"${" + variableCallDigits + "}#",

			&call.Call{
				ID:        uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type:   fmaction.TypeDigitsReceive,
					Option: []byte(`{"length": 3, "key": "#*"}`),
				},
			},
			&variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "#",
				},
			},

			map[string]string{
				variableCallDigits: "${" + variableCallDigits + "}#",
			},
		},
		{
			"finish key *",

			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			"*",
			100,

			"${" + variableCallDigits + "}*",

			&call.Call{
				ID:        uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type:   fmaction.TypeDigitsReceive,
					Option: []byte(`{"length": 3, "key": "#*"}`),
				},
			},
			&variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "*",
				},
			},

			map[string]string{
				variableCallDigits: "${" + variableCallDigits + "}*",
			},
		},
		{
			"finish by max number key 2",

			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			"2",
			100,

			"${" + variableCallDigits + "}2",

			&call.Call{
				ID:        uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type:   fmaction.TypeDigitsReceive,
					Option: []byte(`{"length": 2}`),
				},
			},
			&variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "12",
				},
			},

			map[string]string{
				variableCallDigits: "${" + variableCallDigits + "}2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.responseCall, nil)

			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.responseCall.ActiveFlowID, tt.expectVariables).Return(nil)
			mockReq.EXPECT().FlowV1VariableGet(gomock.Any(), tt.responseCall.ActiveFlowID).Return(tt.responseVariable, nil)
			mockReq.EXPECT().CallV1CallActionNext(gomock.Any(), tt.responseCall.ID, false)

			if err := h.digitsReceived(ctx, tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_DTMFReceived_action_talk_digits_handle_next(t *testing.T) {

	tests := []struct {
		name string

		channel  *channel.Channel
		digit    string
		duration int

		expectDigits string

		responseCall *call.Call

		expectVariables map[string]string
	}{
		{
			"digits",

			&channel.Channel{
				ID:         "c0b5711e-a902-11ed-9f51-c74975f93e22",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			"1",
			100,

			"1",

			&call.Call{
				ID:        uuid.FromStringOrNil("c102c248-a902-11ed-9cdd-439f377ef6a3"),
				ChannelID: "c0b5711e-a902-11ed-9f51-c74975f93e22",
				Action: fmaction.Action{
					Type:   fmaction.TypeTalk,
					Option: []byte(`{"digits_handle": "next"}`),
				},
			},

			map[string]string{
				variableCallDigits: "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.responseCall, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.responseCall.ActiveFlowID, tt.expectVariables).Return(nil)
			mockReq.EXPECT().CallV1CallActionNext(gomock.Any(), tt.responseCall.ID, true)

			if err := h.digitsReceived(ctx, tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_DTMFReceived_action_talk_digits_handle_none(t *testing.T) {

	tests := []struct {
		name string

		channel  *channel.Channel
		digit    string
		duration int

		responseCall *call.Call

		expectDigits    string
		expectVariables map[string]string
	}{
		{
			"digits",

			&channel.Channel{
				ID:         "4f273ce6-a905-11ed-8509-2f79d7c536a1",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			"1",
			100,

			&call.Call{
				ID:        uuid.FromStringOrNil("4f4b048c-a905-11ed-8dfa-07f2bde8ba51"),
				ChannelID: "4f273ce6-a905-11ed-8509-2f79d7c536a1",
				Action: fmaction.Action{
					Type:   fmaction.TypeTalk,
					Option: []byte(`{"digits_handle": ""}`),
				},
			},

			"1",
			map[string]string{
				variableCallDigits: "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.responseCall, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.responseCall.ActiveFlowID, tt.expectVariables).Return(nil)

			if err := h.digitsReceived(ctx, tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_DTMFGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall *call.Call
		responseVar  *variable.Variable

		expectRes string
	}{
		{
			"normal",

			uuid.FromStringOrNil("dec200a8-9014-11ec-9c0b-b35777a9d85a"),

			&call.Call{
				ID:           uuid.FromStringOrNil("1ca7fdc8-defd-11ec-880d-9f8645d0b676"),
				ActiveFlowID: uuid.FromStringOrNil("23989d54-defd-11ec-ae0a-9f577eaf8a74"),
			},
			&variable.Variable{
				ID: uuid.FromStringOrNil("23989d54-defd-11ec-ae0a-9f577eaf8a74"),
				Variables: map[string]string{
					variableCallDigits: "1",
				},
			},

			"1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.responseCall.ActiveFlowID).Return(tt.responseVar, nil)

			res, err := h.DigitsGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_checkDigitsCondition(t *testing.T) {

	tests := []struct {
		name string

		variableID uuid.UUID
		option     *fmaction.OptionDigitsReceive

		responseVariable *variable.Variable

		expectRes bool
	}{
		{
			"length qualified",

			uuid.FromStringOrNil("5578c290-df02-11ec-aa46-c3896c566bef"),
			&fmaction.OptionDigitsReceive{
				Length: 3,
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("8ab35caa-df01-11ec-a567-abb76662ef08"),
				Variables: map[string]string{
					variableCallDigits: "123",
				},
			},

			true,
		},
		{
			"finish on key #",

			uuid.FromStringOrNil("bc06ef06-df01-11ec-ad88-074454252454"),
			&fmaction.OptionDigitsReceive{
				Key: "#",
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("bc06ef06-df01-11ec-ad88-074454252454"),
				Variables: map[string]string{
					variableCallDigits: "#",
				},
			},

			true,
		},
		{
			"finish on key *",

			uuid.FromStringOrNil("e28f7a44-df01-11ec-8eaf-47af6e21909e"),
			&fmaction.OptionDigitsReceive{
				Key: "1234567*",
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("e28f7a44-df01-11ec-8eaf-47af6e21909e"),
				Variables: map[string]string{
					variableCallDigits: "890*",
				},
			},

			true,
		},
		{
			"finish key not match",

			uuid.FromStringOrNil("e28f7a44-df01-11ec-8eaf-47af6e21909e"),
			&fmaction.OptionDigitsReceive{
				Key: "1234567*",
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("e28f7a44-df01-11ec-8eaf-47af6e21909e"),
				Variables: map[string]string{
					variableCallDigits: "089",
				},
			},

			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.variableID).Return(tt.responseVariable, nil)

			res, err := h.checkDigitsCondition(ctx, tt.variableID, tt.option)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
