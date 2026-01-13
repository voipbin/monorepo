package channelhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Answer(t *testing.T) {

	tests := []struct {
		name string

		id string

		responseChannel *channel.Channel
	}{
		{
			"normal",

			"7bb6f7d2-f1d6-11ed-a374-27c4926da333",

			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "7bb6f7d2-f1d6-11ed-a374-27c4926da333",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelAnswer(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID.Return(nil)

			if err := h.Answer(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_DTMFSend(t *testing.T) {

	tests := []struct {
		name string

		id       string
		digit    string
		duration int
		before   int
		between  int
		after    int

		responseChannel *channel.Channel
	}{
		{
			"normal",

			"f982b8d6-f1d6-11ed-a858-273f6fafdf2d",
			"123",
			10,
			11,
			12,
			13,

			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "f982b8d6-f1d6-11ed-a858-273f6fafdf2d",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelDTMF(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.digit, tt.duration, tt.before, tt.between, tt.after.Return(nil)

			if err := h.DTMFSend(ctx, tt.id, tt.digit, tt.duration, tt.before, tt.between, tt.after); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Record(t *testing.T) {

	tests := []struct {
		name string

		id       string
		filename string
		format   string
		duration int
		silence  int
		beep     bool
		endKey   string
		ifExists string

		responseChannel *channel.Channel
	}{
		{
			"normal",

			"7a441186-f1d7-11ed-b7fd-dfeed495bcd9",
			"test_recording_file",
			"wav",
			11,
			12,
			true,
			"#",
			"fail",

			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "7a441186-f1d7-11ed-b7fd-dfeed495bcd9",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelRecord(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.filename, tt.format, tt.duration, tt.silence, tt.beep, tt.endKey, tt.ifExists.Return(nil)

			if err := h.Record(ctx, tt.id, tt.filename, tt.format, tt.duration, tt.silence, tt.beep, tt.endKey, tt.ifExists); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Dial(t *testing.T) {

	tests := []struct {
		name string

		id      string
		caller  string
		timeout int

		responseChannel *channel.Channel
	}{
		{
			"normal",

			"48fa7588-f1d8-11ed-b6fc-33b1a48c2ece",
			"test_caller",
			30,

			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "48fa7588-f1d8-11ed-b6fc-33b1a48c2ece",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelDial(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.caller, tt.timeout.Return(nil)

			if err := h.Dial(ctx, tt.id, tt.caller, tt.timeout); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Redirect(t *testing.T) {

	tests := []struct {
		name string

		id          string
		contextName string
		exten       string
		priority    string

		responseChannel *channel.Channel
	}{
		{
			name: "normal",

			id:          "cf67ff40-f24c-11ed-ac88-67537dac4349",
			contextName: "svc-stasis",
			exten:       "s",
			priority:    "1",

			responseChannel: &channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "cf67ff40-f24c-11ed-ac88-67537dac4349",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstAMIRedirect(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.contextName, tt.exten, tt.priority.Return(nil)

			if err := h.Redirect(ctx, tt.id, tt.contextName, tt.exten, tt.priority); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Continue(t *testing.T) {

	tests := []struct {
		name string

		id          string
		contextName string
		exten       string
		priority    int
		label       string

		responseChannel *channel.Channel
	}{
		{
			name: "normal",

			id:          "cf67ff40-f24c-11ed-ac88-67537dac4349",
			contextName: "svc-stasis",
			exten:       "s",
			priority:    1,
			label:       "",

			responseChannel: &channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "cf67ff40-f24c-11ed-ac88-67537dac4349",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelContinue(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.contextName, tt.exten, tt.priority, tt.label.Return(nil)

			if err := h.Continue(ctx, tt.id, tt.contextName, tt.exten, tt.priority, tt.label); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Ring(t *testing.T) {

	tests := []struct {
		name string

		id string

		responseChannel *channel.Channel
	}{
		{
			name: "normal",

			id: "8da2f10e-f24d-11ed-bc85-0f4baf3d3589",

			responseChannel: &channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "8da2f10e-f24d-11ed-bc85-0f4baf3d3589",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelRing(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID.Return(nil)

			if err := h.Ring(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
