package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmari "monorepo/bin-call-manager/models/ari"
	cmchannel "monorepo/bin-call-manager/models/channel"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AstChannelAnswer(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue  string
		expectURI    string
		expectMethod sock.RequestMethod
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"5734c890-7f6e-11ea-9520-6f774800cd74",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/channels/5734c890-7f6e-11ea-9520-6f774800cd74/answer",
			sock.RequestMethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: "",
					Data:     nil,
				},
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelAnswer(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelContinue(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string
		context    string
		extension  string
		priority   int
		label      string

		expectURI    string
		expectQueue  string
		expectMethod sock.RequestMethod
		expectData   []byte
	}{
		{
			"have all item",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-context",
			"testcall",
			1,
			"testlabel",

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/continue",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodPost,
			[]byte(`{"context":"test-context","extension":"testcall","priority":1,"label":"testlabel"}`),
		},
		{
			"has no label",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-context",
			"testcall",
			1,
			"",

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/continue",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodPost,
			[]byte(`{"context":"test-context","extension":"testcall","priority":1,"label":""}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelContinue(context.Background(), tt.asteriskID, tt.channelID, tt.context, tt.extension, tt.priority, tt.label)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_ChannelAstChannelVariableGet(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string
		variable   string

		response *sock.Response

		expectURI    string
		expectQueue  string
		expectMethod sock.RequestMethod

		expectRes string
	}{
		{
			"have all item",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"value": "test-value"}`),
			},

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable?variable=test-variable",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodGet,

			"test-value",
		},
		{
			"empty value",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"value": ""}`),
			},

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable?variable=test-variable",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodGet,

			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
				},
			).Return(tt.response, nil)

			res, err := reqHandler.AstChannelVariableGet(context.Background(), tt.asteriskID, tt.channelID, tt.variable)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChannelAstChannelVariableSet(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string
		variable   string
		value      string

		expectURI    string
		expectQueue  string
		expectMethod sock.RequestMethod
		expectData   []byte
	}{
		{
			"have all item",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",
			"test-value",

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodPost,

			[]byte(`{"variable":"test-variable","value":"test-value"}`),
		},
		{
			"empty value",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",
			"",

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodPost,
			[]byte(`{"variable":"test-variable","value":""}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelVariableSet(context.Background(), tt.asteriskID, tt.channelID, tt.variable, tt.value)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_ChannelAstChannelHangup(t *testing.T) {

	tests := []struct {
		name        string
		asteriskID  string
		channelID   string
		hangupCause cmari.ChannelCause
		delay       int

		expectURI    string
		expectQueue  string
		expectMethod sock.RequestMethod
		expectData   []byte
	}{
		{
			"have all item",
			"00:11:22:33:44:55",
			"ef6ed35e-828d-11ea-9cd9-83d7b7314faa",
			cmari.ChannelCauseNormalClearing,
			1000,

			"/ari/channels/ef6ed35e-828d-11ea-9cd9-83d7b7314faa",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodDelete,
			[]byte(`{"reason_code":"16"}`),
		},
		{
			"no delay",
			"00:11:22:33:44:55",
			"6fb72ce0-9d54-11ec-b69c-a7c8e3d19337",
			cmari.ChannelCauseNormalClearing,
			0,

			"/ari/channels/6fb72ce0-9d54-11ec-b69c-a7c8e3d19337",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodDelete,
			[]byte(`{"reason_code":"16"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			if tt.delay == 0 {
				mockSock.EXPECT().RequestPublish(
					gomock.Any(),
					tt.expectQueue,
					&sock.Request{
						URI:      tt.expectURI,
						Method:   tt.expectMethod,
						DataType: ContentTypeJSON,
						Data:     tt.expectData,
					},
				).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)
			} else {
				mockSock.EXPECT().RequestPublishWithDelay(
					tt.expectQueue,
					&sock.Request{
						URI:      tt.expectURI,
						Method:   tt.expectMethod,
						DataType: ContentTypeJSON,
						Data:     tt.expectData,
					},
					tt.delay,
				).Return(nil)
			}

			err := reqHandler.AstChannelHangup(context.Background(), tt.asteriskID, tt.channelID, tt.hangupCause, tt.delay)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_ChannelAstChannelCreateSnoop(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string
		snoopID    string
		appArgs    string
		spy        cmchannel.SnoopDirection
		whisper    cmchannel.SnoopDirection

		responseChannel *sock.Response

		expectURI    string
		expectQueue  string
		expectMethod sock.RequestMethod
		expectData   []byte

		expectRes *cmchannel.Channel
	}{
		{
			"have all item",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"test",
			cmchannel.SnoopDirectionIn,
			cmchannel.SnoopDirectionIn,

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"1e5eaa2b-ae8b-412c-b85b-c9d25e70d365"}`),
			},

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodPost,
			[]byte(`{"spy":"in","whisper":"in","app":"voipbin","appArgs":"test","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`),

			&cmchannel.Channel{
				ID:         "1e5eaa2b-ae8b-412c-b85b-c9d25e70d365",
				Data:       map[string]interface{}{},
				StasisData: map[cmchannel.StasisDataType]string{},
			},
		},
		{
			"whisper is none",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"",
			cmchannel.SnoopDirectionIn,
			cmchannel.SnoopDirectionNone,

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"1a367dd8-9635-426c-8f76-3bcafdd71b3c"}`),
			},

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodPost,
			[]byte(`{"spy":"in","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`),

			&cmchannel.Channel{
				ID:         "1a367dd8-9635-426c-8f76-3bcafdd71b3c",
				Data:       map[string]interface{}{},
				StasisData: map[cmchannel.StasisDataType]string{},
			},
		},
		{
			"Spy is none",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"",
			cmchannel.SnoopDirectionNone,
			cmchannel.SnoopDirectionBoth,

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"17dea6c7-ce67-452d-82dd-97afabe6f0ee"}`),
			},

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk.00:11:22:33:44:55.request",
			sock.RequestMethodPost,
			[]byte(`{"whisper":"both","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`),

			&cmchannel.Channel{
				ID:         "17dea6c7-ce67-452d-82dd-97afabe6f0ee",
				Data:       map[string]interface{}{},
				StasisData: map[cmchannel.StasisDataType]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(tt.responseChannel, nil)

			res, err := reqHandler.AstChannelCreateSnoop(ctx, tt.asteriskID, tt.channelID, tt.snoopID, tt.appArgs, tt.spy, tt.whisper)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AstChannelGet(t *testing.T) {

	tests := []struct {
		name     string
		asterisk string
		id       string
		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request

		expectChannel *cmchannel.Channel
	}{
		{
			"normal test",
			"00:11:22:33:44:55",
			"1589711094.100",
			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"1589711094.100","name":"PJSIP/call-in-00000019","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"8872616","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=xt1GqgsEfG,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-17T10:24:54.396+0000","language":"en"}`),
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/1589711094.100",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     nil,
			},
			&cmchannel.Channel{
				ID:         "1589711094.100",
				AsteriskID: "",
				Name:       "PJSIP/call-in-00000019",
				Tech:       cmchannel.TechPJSIP,

				SourceName:   "tttt",
				SourceNumber: "pchero",

				DestinationNumber: "8872616",
				State:             cmari.ChannelStateUp,

				Data:       map[string]interface{}{},
				StasisData: map[cmchannel.StasisDataType]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AstChannelGet(context.Background(), tt.asterisk, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, res)
			}
		})
	}

}

func Test_AstChannelDTMF(t *testing.T) {

	tests := []struct {
		name     string
		asterisk string
		id       string
		digit    string
		duration int
		before   int
		between  int
		after    int
		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			"normal test",
			"00:11:22:33:44:55",
			"6d11e7c2-9a69-11ea-95af-eb4a15c08df1",
			"1",
			100,
			0,
			0,
			0,
			&sock.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/6d11e7c2-9a69-11ea-95af-eb4a15c08df1/dtmf",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"dtmf":"1","duration":100,"before":0,"between":0,"after":0}`),
			},
		},
		{
			"longer digits",
			"00:11:22:33:44:55",
			"6d11e7c2-9a69-11ea-95af-eb4a15c08df1",
			"19827348",
			100,
			0,
			0,
			0,
			&sock.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/6d11e7c2-9a69-11ea-95af-eb4a15c08df1/dtmf",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"dtmf":"19827348","duration":100,"before":0,"between":0,"after":0}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelDTMF(context.Background(), tt.asterisk, tt.id, tt.digit, tt.duration, tt.before, tt.between, tt.after)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelCreate(t *testing.T) {

	tests := []struct {
		name           string
		asterisk       string
		channelID      string
		appArgs        string
		endpoint       string
		otherChannelID string
		originator     string
		formats        string
		variables      map[string]string
		response       *sock.Response

		expectTarget  string
		expectRequest *sock.Request

		expectRes *cmchannel.Channel
	}{
		{
			"normal test",
			"00:11:22:33:44:55",
			"adf2ec1a-9ee6-11ea-9d2e-33da3e3b92a3",
			"",
			"PJSIP/call-out/sip:test@test.com:5060",
			"",
			"",
			"",
			nil,
			&sock.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"e28258f0-4267-475c-99a3-348bc580f9dd"}`),
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/create",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"endpoint":"PJSIP/call-out/sip:test@test.com:5060","app":"voipbin","channelId":"adf2ec1a-9ee6-11ea-9d2e-33da3e3b92a3"}`),
			},

			&cmchannel.Channel{
				ID:         "e28258f0-4267-475c-99a3-348bc580f9dd",
				Data:       map[string]interface{}{},
				StasisData: map[cmchannel.StasisDataType]string{},
			},
		},
		{
			"include variables",
			"00:11:22:33:44:55",
			"0a2628dc-0853-11eb-811f-ebaf03ba1ba6",
			"",
			"PJSIP/call-out/sip:test@test.com:5060",
			"",
			"",
			"",
			map[string]string{"CALLERID(all)": "+123456789"},
			&sock.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"186d1169-ff9a-4b91-84cd-011585a63dc4"}`),
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/create",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"endpoint":"PJSIP/call-out/sip:test@test.com:5060","app":"voipbin","channelId":"0a2628dc-0853-11eb-811f-ebaf03ba1ba6","variables":{"CALLERID(all)":"+123456789"}}`),
			},

			&cmchannel.Channel{
				ID:         "186d1169-ff9a-4b91-84cd-011585a63dc4",
				Data:       map[string]interface{}{},
				StasisData: map[cmchannel.StasisDataType]string{},
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AstChannelCreate(context.Background(), tt.asterisk, tt.channelID, tt.appArgs, tt.endpoint, tt.otherChannelID, tt.originator, tt.formats, tt.variables)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AstChannelDial(t *testing.T) {

	tests := []struct {
		name      string
		asterisk  string
		channelID string
		caller    string
		timeout   int
		response  *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			"empty caller",
			"00:11:22:33:44:55",
			"83a188ba-a060-11ea-a777-038b061dfbc3",
			"",
			30,
			&sock.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/83a188ba-a060-11ea-a777-038b061dfbc3/dial",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"timeout":30}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelDial(context.Background(), tt.asterisk, tt.channelID, tt.caller, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelPlay(t *testing.T) {

	tests := []struct {
		name      string
		asterisk  string
		channelID string
		actionID  uuid.UUID
		medias    []string
		response  *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			"1 media",
			"00:11:22:33:44:55",
			"94bcc2b4-e718-11ea-a8cf-e7d1a61482a8",
			uuid.FromStringOrNil("c44864cc-e7d9-11ea-923a-73e96775044d"),
			[]string{"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"},
			&sock.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/94bcc2b4-e718-11ea-a8cf-e7d1a61482a8/play",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"media":["sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"],"playbackId":"c44864cc-e7d9-11ea-923a-73e96775044d"}`),
			},
		},
		{
			"2 medias",
			"00:11:22:33:44:55",
			"94bcc2b4-e718-11ea-a8cf-e7d1a61482a8",
			uuid.FromStringOrNil("dde1c518-e7d9-11ea-902a-2b04669d8a49"),
			[]string{"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav", "sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm-test.wav"},
			&sock.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/94bcc2b4-e718-11ea-a8cf-e7d1a61482a8/play",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"media":["sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm-test.wav"],"playbackId":"dde1c518-e7d9-11ea-902a-2b04669d8a49"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelPlay(context.Background(), tt.asterisk, tt.channelID, tt.actionID, tt.medias, "")
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelRecord(t *testing.T) {

	tests := []struct {
		name      string
		asterisk  string
		channelID string
		filename  string
		format    string
		duration  int
		silence   int
		beep      bool
		endKey    string
		ifExist   string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"b3b6ca04-28d4-11eb-a27e-ebcd6dfed523",
			"call_b469f3cc-28d4-11eb-b29a-db389e2bf1ca_2020-05-17T10:24:54.396+0000",
			"wav",
			0,
			0,
			false,
			"",
			"fail",

			&sock.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/b3b6ca04-28d4-11eb-a27e-ebcd6dfed523/record",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"call_b469f3cc-28d4-11eb-b29a-db389e2bf1ca_2020-05-17T10:24:54.396+0000","format":"wav","maxDurationSeconds":0,"maxSilenceSeconds":0,"beep":false,"terminateOn":"","ifExists":"fail"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelRecord(context.Background(), tt.asterisk, tt.channelID, tt.filename, tt.format, tt.duration, tt.silence, tt.beep, tt.endKey, tt.ifExist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelExternalMedia(t *testing.T) {

	tests := []struct {
		name string

		asterisk       string
		channelID      string
		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string
		data           string
		variables      map[string]string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cmchannel.Channel
	}{
		{
			"normal",

			"00:11:22:33:44:55",
			"660486e8-ffca-11eb-aef3-6b2e6caea5a5",
			"http://test.com/external-sample",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",
			"",
			nil,

			&sock.Response{
				StatusCode: 200,
				Data:       []byte(`{"id": "660486e8-ffca-11eb-aef3-6b2e6caea5a5","name": "UnicastRTP/127.0.0.1:5090-0x7f6d54035300","state": "Down","caller": {"name": "","number": ""},"connected": {"name": "","number": ""},"accountcode": "","dialplan": {"context": "default","exten": "s","priority": 1,"app_name": "AppDial2","app_data": "(Outgoing Line)"},"creationtime": "2021-08-22T04:10:10.331+0000","language": "en","channelvars": {"UNICASTRTP_LOCAL_PORT": "10492","UNICASTRTP_LOCAL_ADDRESS": "127.0.0.1"}}`),
			},

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/externalMedia",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"channel_id":"660486e8-ffca-11eb-aef3-6b2e6caea5a5","app":"voipbin","external_host":"http://test.com/external-sample","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},
			&cmchannel.Channel{
				ID:                "660486e8-ffca-11eb-aef3-6b2e6caea5a5",
				Name:              "UnicastRTP/127.0.0.1:5090-0x7f6d54035300",
				Tech:              cmchannel.TechUnicatRTP,
				DestinationNumber: "s",
				State:             cmari.ChannelStateDown,
				Data: map[string]interface{}{
					"UNICASTRTP_LOCAL_ADDRESS": "127.0.0.1",
					"UNICASTRTP_LOCAL_PORT":    "10492",
				},
				StasisData: map[cmchannel.StasisDataType]string{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AstChannelExternalMedia(context.Background(), tt.asterisk, tt.channelID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction, tt.data, tt.variables)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_AstChannelRing(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue  string
		expectURI    string
		expectMethod sock.RequestMethod
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"519b7e6a-9790-11ec-ae44-23af21dc0b55",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/channels/519b7e6a-9790-11ec-ae44-23af21dc0b55/ring",
			sock.RequestMethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: "application/json",
					Data:     nil,
				},
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelRing(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelHoldOn(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"ef557106-ced0-11ed-83d5-8f5e5a3bd8a7",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:    "/ari/channels/ef557106-ced0-11ed-83d5-8f5e5a3bd8a7/hold",
				Method: sock.RequestMethodPost,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelHoldOn(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelHoldOff(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"3c1a71f8-ced1-11ed-a2b9-0fca69fc6f49",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:    "/ari/channels/3c1a71f8-ced1-11ed-a2b9-0fca69fc6f49/hold",
				Method: sock.RequestMethodDelete,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelHoldOff(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelMusicOnHoldOn(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"d7602848-d0b5-11ed-8240-afb9293d0c44",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:    "/ari/channels/d7602848-d0b5-11ed-8240-afb9293d0c44/moh",
				Method: sock.RequestMethodPost,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelMusicOnHoldOn(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelMusicOnHoldOff(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"d795433e-d0b5-11ed-9884-9f7d67594400",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:    "/ari/channels/d795433e-d0b5-11ed-9884-9f7d67594400/moh",
				Method: sock.RequestMethodDelete,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelMusicOnHoldOff(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelSilenceOn(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"d7c3b2c8-d0b5-11ed-b0ca-835a116c1710",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:    "/ari/channels/d7c3b2c8-d0b5-11ed-b0ca-835a116c1710/silence",
				Method: sock.RequestMethodPost,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelSilenceOn(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelSilenceOff(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"d7f2b208-d0b5-11ed-bddd-b796b66c6d5f",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:    "/ari/channels/d7f2b208-d0b5-11ed-bddd-b796b66c6d5f/silence",
				Method: sock.RequestMethodDelete,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelSilenceOff(context.Background(), tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelMuteOn(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string
		direction  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"d8393e4e-d0b5-11ed-9a0a-6fd4e76c9f86",
			"both",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/d8393e4e-d0b5-11ed-9a0a-6fd4e76c9f86/mute",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"direction":"both"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelMuteOn(context.Background(), tt.asteriskID, tt.channelID, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstChannelMuteOff(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string
		direction  string

		expectQueue   string
		expectRequest *sock.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"4544a38e-d0b6-11ed-beab-13e97aee6eb0",
			"both",

			"asterisk.00:11:22:33:44:55.request",
			&sock.Request{
				URI:      "/ari/channels/4544a38e-d0b6-11ed-beab-13e97aee6eb0/mute",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"direction":"both"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				tt.expectRequest,
			).Return(&sock.Response{StatusCode: 200, Data: nil}, nil)

			err := reqHandler.AstChannelMuteOff(context.Background(), tt.asteriskID, tt.channelID, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}
