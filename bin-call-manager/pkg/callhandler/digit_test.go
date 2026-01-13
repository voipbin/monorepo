package callhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/dtmf"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_digitsReceivedNotActionDTMFReceived(t *testing.T) {

	tests := []struct {
		name     string
		channel  *channel.Channel
		digit    string
		duration int

		responseCall       *call.Call
		responseUUIDDTMFID uuid.UUID
		responseCurTime    string
		expectDTMF         *dtmf.DTMF
		expectVariables    map[string]string
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				ID:         "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			digit:    "4",
			duration: 100,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2a45cf6-9ace-11ea-9354-4baa7f3ad331"),
					CustomerID: uuid.FromStringOrNil("c098148c-b838-11f0-a16e-8ba6f1e91be8"),
				},
				ChannelID: "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				Action: fmaction.Action{
					Type: fmaction.TypeEcho,
				},
			},
			responseUUIDDTMFID: uuid.FromStringOrNil("f496f2bc-b838-11f0-a757-4b893b2a9030"),
			responseCurTime:    "2020-04-18 05:22:17.995000",
			expectDTMF: &dtmf.DTMF{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f496f2bc-b838-11f0-a757-4b893b2a9030"),
					CustomerID: uuid.FromStringOrNil("c098148c-b838-11f0-a16e-8ba6f1e91be8"),
				},
				CallID:   uuid.FromStringOrNil("b2a45cf6-9ace-11ea-9354-4baa7f3ad331"),
				Digit:    "4",
				Duration: 100,

				TMCreate: "2020-04-18 05:22:17.995000",
			},
			expectVariables: map[string]string{
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(ctx, tt.channel.ID.Return(tt.responseCall, nil)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDDTMFID)
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockNotify.EXPECT().PublishEvent(ctx, dtmf.EventTypeDTMFReceived, tt.expectDTMF)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseCall.ActiveflowID, tt.expectVariables.Return(nil)

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

		responseCall       *call.Call
		responseUUIDDTMFID uuid.UUID
		responseCurTime    string
		responseVar        *variable.Variable
		savedDTMFs         string

		expectDTMF      *dtmf.DTMF
		expectVariables map[string]string
	}{
		{
			name: "length not enough",
			channel: &channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			digit:    "4",
			duration: 100,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				},
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					Option: map[string]any{
						"length": 3,
					},
				},
			},
			responseUUIDDTMFID: uuid.FromStringOrNil("edf469a2-b870-11f0-b28c-dfe1694e1cbe"),
			responseCurTime:    "2020-04-18 05:22:17.995000",
			responseVar: &variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "",
				},
			},
			savedDTMFs: "",

			expectDTMF: &dtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("edf469a2-b870-11f0-b28c-dfe1694e1cbe"),
				},
				CallID:   uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				Digit:    "4",
				Duration: 100,
				TMCreate: "2020-04-18 05:22:17.995000",
			},
			expectVariables: map[string]string{
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(ctx, tt.channel.ID.Return(tt.responseCall, nil)

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDDTMFID)
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockNotify.EXPECT().PublishEvent(ctx, dtmf.EventTypeDTMFReceived, tt.expectDTMF)

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.responseCall.ActiveflowID.Return(tt.responseVar, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseCall.ActiveflowID, tt.expectVariables.Return(nil)

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

		responseCall       *call.Call
		responseUUIDDTMFID uuid.UUID
		responseCurTime    string

		responseVariable *variable.Variable

		expectDigits    string
		expectDTMF      *dtmf.DTMF
		expectVariables map[string]string
	}{
		{
			name: "finish key #",

			channel: &channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			digit:    "#",
			duration: 100,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				},
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					Option: map[string]any{
						"length": 3,
						"key":    "#*",
					},
				},
			},
			responseUUIDDTMFID: uuid.FromStringOrNil("abd9ce8a-b871-11f0-a7e7-0b3034922233"),
			responseCurTime:    "2020-04-18 05:22:17.995000",
			responseVariable: &variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "#",
				},
			},

			expectDigits: "${" + variableCallDigits + "}#",
			expectDTMF: &dtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("abd9ce8a-b871-11f0-a7e7-0b3034922233"),
				},
				CallID:   uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				Digit:    "#",
				Duration: 100,
				TMCreate: "2020-04-18 05:22:17.995000",
			},
			expectVariables: map[string]string{
				variableCallDigits: "${" + variableCallDigits + "}#",
			},
		},
		{
			name: "finish key *",

			channel: &channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			digit:    "*",
			duration: 100,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				},
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					Option: map[string]any{
						"length": 3,
						"key":    "#*",
					},
				},
			},
			responseUUIDDTMFID: uuid.FromStringOrNil("ac10d5ba-b871-11f0-838c-0bceb98efacf"),
			responseCurTime:    "2020-04-18 05:22:17.995000",
			responseVariable: &variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "*",
				},
			},

			expectDigits: "${" + variableCallDigits + "}*",
			expectDTMF: &dtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ac10d5ba-b871-11f0-838c-0bceb98efacf"),
				},
				CallID:   uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				Digit:    "*",
				Duration: 100,
				TMCreate: "2020-04-18 05:22:17.995000",
			},
			expectVariables: map[string]string{
				variableCallDigits: "${" + variableCallDigits + "}*",
			},
		},
		{
			name: "finish by max number key 2",

			channel: &channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			digit:    "2",
			duration: 100,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				},
				ChannelID: "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					Option: map[string]any{
						"length": 2,
					},
				},
			},
			responseUUIDDTMFID: uuid.FromStringOrNil("ac352b5e-b871-11f0-90f4-7fffb7fdc179"),
			responseCurTime:    "2020-04-18 05:22:17.995000",
			responseVariable: &variable.Variable{
				Variables: map[string]string{
					variableCallDigits: "12",
				},
			},

			expectDigits: "${" + variableCallDigits + "}2",
			expectDTMF: &dtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ac352b5e-b871-11f0-90f4-7fffb7fdc179"),
				},
				CallID:   uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				Digit:    "2",
				Duration: 100,
				TMCreate: "2020-04-18 05:22:17.995000",
			},
			expectVariables: map[string]string{
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID.Return(tt.responseCall, nil)

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDDTMFID)
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), dtmf.EventTypeDTMFReceived, tt.expectDTMF)

			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.responseCall.ActiveflowID, tt.expectVariables.Return(nil)
			mockReq.EXPECT().FlowV1VariableGet(gomock.Any(), tt.responseCall.ActiveflowID.Return(tt.responseVariable, nil)
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

		responseCall       *call.Call
		responseUUIDDTMFID uuid.UUID
		responseCurTime    string

		expectDigits    string
		expectDTMF      *dtmf.DTMF
		expectVariables map[string]string
	}{
		{
			name: "digits",

			channel: &channel.Channel{
				ID:         "c0b5711e-a902-11ed-9f51-c74975f93e22",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			digit:    "1",
			duration: 100,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c102c248-a902-11ed-9cdd-439f377ef6a3"),
				},
				ChannelID: "c0b5711e-a902-11ed-9f51-c74975f93e22",
				Action: fmaction.Action{
					Type: fmaction.TypeTalk,
					Option: map[string]any{
						"digits_handle": "next",
					},
				},
			},
			responseUUIDDTMFID: uuid.FromStringOrNil("6113fd7a-b872-11f0-ab9a-6f891f023f67"),
			responseCurTime:    "2020-04-18 05:22:17.995000",

			expectDigits: "1",
			expectDTMF: &dtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6113fd7a-b872-11f0-ab9a-6f891f023f67"),
				},
				CallID:   uuid.FromStringOrNil("c102c248-a902-11ed-9cdd-439f377ef6a3"),
				Digit:    "1",
				Duration: 100,
				TMCreate: "2020-04-18 05:22:17.995000",
			},
			expectVariables: map[string]string{
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID.Return(tt.responseCall, nil)

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDDTMFID)
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), dtmf.EventTypeDTMFReceived, tt.expectDTMF)

			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.responseCall.ActiveflowID, tt.expectVariables.Return(nil)
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

		responseCall       *call.Call
		responseUUIDDTMFID uuid.UUID
		responseCurTime    string

		expectDigits    string
		expectDTMF      *dtmf.DTMF
		expectVariables map[string]string
	}{
		{
			name: "digits",

			channel: &channel.Channel{
				ID:         "4f273ce6-a905-11ed-8509-2f79d7c536a1",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			digit:    "1",
			duration: 100,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f4b048c-a905-11ed-8dfa-07f2bde8ba51"),
				},
				ChannelID: "4f273ce6-a905-11ed-8509-2f79d7c536a1",
				Action: fmaction.Action{
					Type: fmaction.TypeTalk,
					Option: map[string]any{
						"digits_handle": "",
					},
				},
			},
			responseUUIDDTMFID: uuid.FromStringOrNil("f378c538-b872-11f0-8203-8fe3f320a39a"),
			responseCurTime:    "2020-04-18 05:22:17.995000",

			expectDigits: "1",
			expectDTMF: &dtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f378c538-b872-11f0-8203-8fe3f320a39a"),
				},
				CallID:   uuid.FromStringOrNil("4f4b048c-a905-11ed-8dfa-07f2bde8ba51"),
				Digit:    "1",
				Duration: 100,
				TMCreate: "2020-04-18 05:22:17.995000",
			},
			expectVariables: map[string]string{
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID.Return(tt.responseCall, nil)

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDDTMFID)
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), dtmf.EventTypeDTMFReceived, tt.expectDTMF)

			mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), tt.responseCall.ActiveflowID, tt.expectVariables.Return(nil)

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1ca7fdc8-defd-11ec-880d-9f8645d0b676"),
				},
				ActiveflowID: uuid.FromStringOrNil("23989d54-defd-11ec-ae0a-9f577eaf8a74"),
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

			mockDB.EXPECT().CallGet(ctx, tt.id.Return(tt.responseCall, nil)
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.responseCall.ActiveflowID.Return(tt.responseVar, nil)

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

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.variableID.Return(tt.responseVariable, nil)

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
