package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestDTMFReceivedNotActionDTMFReceived(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name     string
		channel  *channel.Channel
		call     *call.Call
		digit    string
		duration int
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("b2a45cf6-9ace-11ea-9354-4baa7f3ad331"),
				ChannelID:  "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
				Action: action.Action{
					Type: action.TypeEcho,
				},
			},
			"4",
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallDTMFSet(gomock.Any(), tt.call.ID, tt.digit).Return(nil)

			if err := h.DTMFReceived(tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestDTMFReceivedContinue(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		call       *call.Call
		digit      string
		duration   int
		savedDTMFs string
	}

	tests := []test{
		{
			"max num is 3, got 1",
			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID:  "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
				Action: action.Action{
					Type:   action.TypeDTMFReceive,
					Option: []byte(`{"max_number_key": 3}`),
				},
			},
			"4",
			100,
			"",
		},
		{
			"max num is 3, got 2",
			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID:  "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
				Action: action.Action{
					Type:   action.TypeDTMFReceive,
					Option: []byte(`{"max_number_key": 3}`),
				},
			},
			"4",
			100,
			"1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallDTMFGet(gomock.Any(), tt.call.ID).Return(tt.savedDTMFs, nil)
			mockDB.EXPECT().CallDTMFSet(gomock.Any(), tt.call.ID, tt.savedDTMFs+tt.digit).Return(nil)

			if err := h.DTMFReceived(tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestDTMFReceivedStop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		call       *call.Call
		digit      string
		duration   int
		savedDTMFs string
	}

	tests := []test{
		{
			"finish key #",
			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID:  "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
				Action: action.Action{
					Type:   action.TypeDTMFReceive,
					Option: []byte(`{"max_number_key": 3, "finish_on_key": "#*"}`),
				},
			},
			"#",
			100,
			"",
		},
		{
			"finish key *",
			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID:  "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
				Action: action.Action{
					Type:   action.TypeDTMFReceive,
					Option: []byte(`{"max_number_key": 3, "finish_on_key": "#*"}`),
				},
			},
			"#",
			100,
			"1",
		},
		{
			"finish by max number key 2",
			&channel.Channel{
				ID:         "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("f0f0f6bc-695a-11eb-ae99-0b10f2bf1b94"),
				ChannelID:  "f7ac13c4-695a-11eb-aba7-7f6e7457f0b8",
				AsteriskID: "80:fa:5b:5e:da:81",
				Action: action.Action{
					Type:   action.TypeDTMFReceive,
					Option: []byte(`{"max_number_key": 2}`),
				},
			},
			"2",
			100,
			"1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallDTMFGet(gomock.Any(), tt.call.ID).Return(tt.savedDTMFs, nil)
			mockDB.EXPECT().CallDTMFSet(gomock.Any(), tt.call.ID, tt.savedDTMFs+tt.digit).Return(nil)
			mockReq.EXPECT().CMV1CallActionNext(gomock.Any(), tt.call.ID)

			if err := h.DTMFReceived(tt.channel, tt.digit, tt.duration); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
