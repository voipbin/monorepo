package channelhandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_StartSnoop(t *testing.T) {

	type test struct {
		name string

		id               string
		snoopID          string
		appArgs          string
		directionSpy     channel.SnoopDirection
		directionWhisper channel.SnoopDirection

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			id:               "d2d4036e-f2a5-11ed-9338-4fcdc112773e",
			snoopID:          "d343fa7a-f2a5-11ed-87fe-e735418cd66f",
			appArgs:          "testargs=test",
			directionSpy:     channel.SnoopDirectionIn,
			directionWhisper: channel.SnoopDirectionOut,

			responseChannel: &channel.Channel{
				ID:         "d2d4036e-f2a5-11ed-9338-4fcdc112773e",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
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
			mockReq.EXPECT().AstChannelCreateSnoop(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.snoopID, tt.appArgs, tt.directionSpy, tt.directionWhisper.Return(tt.responseChannel, nil)

			res, err := h.StartSnoop(ctx, tt.id, tt.snoopID, tt.appArgs, tt.directionSpy, tt.directionWhisper)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseChannel, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}

func Test_StartExternalMedia(t *testing.T) {

	type test struct {
		name string

		asteriskID     string
		id             string
		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string
		data           string
		variables      map[string]string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			asteriskID:     "3e:50:6b:43:bb:30",
			id:             "d2d4036e-f2a5-11ed-9338-4fcdc112773e",
			externalHost:   "example.com",
			encapsulation:  "rtp",
			transport:      "udp",
			connectionType: "client",
			format:         "ulaw",
			direction:      "both",
			data:           "context=call-externalmedia",
			variables: map[string]string{
				"key1": "value1",
			},

			responseChannel: &channel.Channel{
				ID:         "d2d4036e-f2a5-11ed-9338-4fcdc112773e",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
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

			mockReq.EXPECT().AstChannelExternalMedia(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction, tt.data, tt.variables.Return(tt.responseChannel, nil)

			res, err := h.StartExternalMedia(ctx, tt.asteriskID, tt.id, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction, tt.data, tt.variables)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseChannel, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}

func Test_StartChannel(t *testing.T) {

	type test struct {
		name string

		asteriskID     string
		id             string
		appArgs        string
		endpoint       string
		otherChannelID string
		originator     string
		formats        string
		variables      map[string]string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			asteriskID:     "3e:50:6b:43:bb:30",
			id:             "c6242ba0-f2a8-11ed-89d3-d30565f7c859",
			appArgs:        "test=testarg",
			endpoint:       "pjsip/call-out/sip:testoutgoing@test.com",
			otherChannelID: "c6615408-f2a8-11ed-a679-e3d2a2222397",
			originator:     "test_originator",
			formats:        "ulaw",
			variables: map[string]string{
				"key1": "value1",
			},

			responseChannel: &channel.Channel{
				ID:         "d2d4036e-f2a5-11ed-9338-4fcdc112773e",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
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

			mockReq.EXPECT().AstChannelCreate(ctx, tt.asteriskID, tt.id, tt.appArgs, tt.endpoint, tt.otherChannelID, tt.originator, tt.formats, tt.variables.Return(tt.responseChannel, nil)

			res, err := h.StartChannel(ctx, tt.asteriskID, tt.id, tt.appArgs, tt.endpoint, tt.otherChannelID, tt.originator, tt.formats, tt.variables)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseChannel, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}

func Test_StartChannelWithBaseChannel(t *testing.T) {

	type test struct {
		name string

		baseChannelID  string
		id             string
		appArgs        string
		endpoint       string
		otherChannelID string
		originator     string
		formats        string
		variables      map[string]string

		responseBaseChannel *channel.Channel
		responseChannel     *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			baseChannelID:  "25ef5bb2-f2aa-11ed-a779-1b07c9d70cca",
			id:             "261a4688-f2aa-11ed-a10b-bbd267479090",
			appArgs:        "test=testarg",
			endpoint:       "pjsip/call-out/sip:testoutgoing@test.com",
			otherChannelID: "c6615408-f2a8-11ed-a679-e3d2a2222397",
			originator:     "test_originator",
			formats:        "ulaw",
			variables: map[string]string{
				"key1": "value1",
			},

			responseBaseChannel: &channel.Channel{
				ID:         "25ef5bb2-f2aa-11ed-a779-1b07c9d70cca",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseChannel: &channel.Channel{
				ID:         "261a4688-f2aa-11ed-a10b-bbd267479090",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
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

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.baseChannelID.Return(tt.responseBaseChannel, nil)
			mockReq.EXPECT().AstChannelCreate(ctx, tt.responseBaseChannel.AsteriskID, tt.id, tt.appArgs, tt.endpoint, tt.otherChannelID, tt.originator, tt.formats, tt.variables.Return(tt.responseChannel, nil)

			res, err := h.StartChannelWithBaseChannel(ctx, tt.baseChannelID, tt.id, tt.appArgs, tt.endpoint, tt.otherChannelID, tt.originator, tt.formats, tt.variables)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseChannel, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}
