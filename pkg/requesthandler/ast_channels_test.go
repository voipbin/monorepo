package requesthandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// func TestSetSock(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSock := rabbitmq.NewMockRabbit(mc)
// 	if mockSock == nil {
// 		t.Errorf("Error")
// 	}

// 	reqHandler := NewRequestHandler(mockSock)

// 	if reqHandler == nil {
// 		t.Errorf("Wrong match. expact: true, got: false")
// 	}
// }

func TestAstChannelAnswer(t *testing.T) {

	type test struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmq.RequestMethod
	}

	tests := []test{
		{
			"normal",
			"00:11:22:33:44:55",
			"5734c890-7f6e-11ea-9520-6f774800cd74",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/channels/5734c890-7f6e-11ea-9520-6f774800cd74/answer",
			rabbitmq.RequestMethodPost,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmq.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: "",
					Data:     "",
				},
			).Return(&rabbitmq.Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.AstChannelAnswer(tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func TestAstChannelContinue(t *testing.T) {

	type test struct {
		name       string
		asteriskID string
		channelID  string
		context    string
		extension  string
		priority   int
		label      string

		expectURI    string
		expectQueue  string
		expectMethod rabbitmq.RequestMethod
		expectData   string
	}

	tests := []test{
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
			rabbitmq.RequestMethodPost,
			`{"context":"test-context","extension":"testcall","priority":1,"label":"testlabel"}`,
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
			rabbitmq.RequestMethodPost,
			`{"context":"test-context","extension":"testcall","priority":1,"label":""}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmq.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmq.Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.AstChannelContinue(tt.asteriskID, tt.channelID, tt.context, tt.extension, tt.priority, tt.label)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func TestChannelAstChannelVariableSet(t *testing.T) {

	type test struct {
		name       string
		asteriskID string
		channelID  string
		variable   string
		value      string

		expectURI    string
		expectQueue  string
		expectMethod rabbitmq.RequestMethod
		expectData   string
	}

	tests := []test{
		{
			"have all item",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",
			"test-value",

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable",
			"asterisk.00:11:22:33:44:55.request",
			rabbitmq.RequestMethodPost,

			`{"variable":"test-variable","value":"test-value"}`,
		},
		{
			"empty value",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",
			"",

			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable",
			"asterisk.00:11:22:33:44:55.request",
			rabbitmq.RequestMethodPost,
			`{"variable":"test-variable","value":""}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmq.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmq.Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.AstChannelVariableSet(tt.asteriskID, tt.channelID, tt.variable, tt.value)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func TestChannelAstChannelHangup(t *testing.T) {

	type test struct {
		name        string
		asteriskID  string
		channelID   string
		hangupCause ari.ChannelCause

		expectURI    string
		expectQueue  string
		expectMethod rabbitmq.RequestMethod
		expectData   string
	}

	tests := []test{
		{
			"have all item",
			"00:11:22:33:44:55",
			"ef6ed35e-828d-11ea-9cd9-83d7b7314faa",
			ari.ChannelCauseNormalClearing,

			"/ari/channels/ef6ed35e-828d-11ea-9cd9-83d7b7314faa",
			"asterisk.00:11:22:33:44:55.request",
			rabbitmq.RequestMethodDelete,
			`{"reason_code":"16"}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmq.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmq.Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.AstChannelHangup(tt.asteriskID, tt.channelID, tt.hangupCause)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func TestChannelAstChannelCreateSnoop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	type test struct {
		name       string
		asteriskID string
		channelID  string
		snoopID    string
		appArgs    string
		spy        channel.SnoopDirection
		whisper    channel.SnoopDirection

		expectURI    string
		expectQueue  string
		expectMethod rabbitmq.RequestMethod
		expectData   string
	}

	tests := []test{
		{
			"have all item",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"test",
			channel.SnoopDirectionIn,
			channel.SnoopDirectionIn,

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk.00:11:22:33:44:55.request",
			rabbitmq.RequestMethodPost,
			`{"spy":"in","whisper":"in","app":"voipbin","appArgs":"test","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`,
		},
		{
			"whisper is none",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"",
			channel.SnoopDirectionIn,
			channel.SnoopDirectionNone,

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk.00:11:22:33:44:55.request",
			rabbitmq.RequestMethodPost,
			`{"spy":"in","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`,
		},
		{
			"Spy is none",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"",
			channel.SnoopDirectionNone,
			channel.SnoopDirectionBoth,

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk.00:11:22:33:44:55.request",
			rabbitmq.RequestMethodPost,
			`{"whisper":"both","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmq.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmq.Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.AstChannelCreateSnoop(tt.asteriskID, tt.channelID, tt.snoopID, tt.appArgs, tt.spy, tt.whisper)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func TestAstChannelGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	type test struct {
		name     string
		asterisk string
		id       string
		response *rabbitmq.Response

		expectTarget  string
		expectRequest *rabbitmq.Request

		expectURI     string
		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"normal test",
			"00:11:22:33:44:55",
			"1589711094.100",
			&rabbitmq.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       `{"id":"1589711094.100","name":"PJSIP/call-in-00000019","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"8872616","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=xt1GqgsEfG,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-17T10:24:54.396+0000","language":"en"}`,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmq.Request{
				URI:      "/ari/channels/1589711094.100",
				Method:   rabbitmq.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     "",
			},
			"/ari/channels/1589711094.100",
			&channel.Channel{
				ID:         "1589711094.100",
				AsteriskID: "",
				Name:       "PJSIP/call-in-00000019",
				Tech:       channel.TechPJSIP,

				SourceName:   "tttt",
				SourceNumber: "pchero",

				DestinationNumber: "8872616",
				State:             ari.ChannelStateUp,

				Data: map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AstChannelGet(tt.asterisk, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, res)
			}
		})
	}

}

func TestAstChannelDTMF(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	type test struct {
		name     string
		asterisk string
		id       string
		digit    string
		duration int
		before   int
		between  int
		after    int
		response *rabbitmq.Response

		expectTarget  string
		expectRequest *rabbitmq.Request
	}

	tests := []test{
		{
			"normal test",
			"00:11:22:33:44:55",
			"6d11e7c2-9a69-11ea-95af-eb4a15c08df1",
			"1",
			100,
			0,
			0,
			0,
			&rabbitmq.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmq.Request{
				URI:      "/ari/channels/6d11e7c2-9a69-11ea-95af-eb4a15c08df1/dtmf",
				Method:   rabbitmq.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     `{"dtmf":"1","duration":100,"before":0,"between":0,"after":0}`,
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
			&rabbitmq.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmq.Request{
				URI:      "/ari/channels/6d11e7c2-9a69-11ea-95af-eb4a15c08df1/dtmf",
				Method:   rabbitmq.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     `{"dtmf":"19827348","duration":100,"before":0,"between":0,"after":0}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelDTMF(tt.asterisk, tt.id, tt.digit, tt.duration, tt.before, tt.between, tt.after)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAstChannelCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	type test struct {
		name           string
		asterisk       string
		chanelID       string
		appArgs        string
		endpoint       string
		otherChannelID string
		originator     string
		formats        string
		response       *rabbitmq.Response

		expectTarget  string
		expectRequest *rabbitmq.Request
	}

	tests := []test{
		{
			"normal test",
			"00:11:22:33:44:55",
			"adf2ec1a-9ee6-11ea-9d2e-33da3e3b92a3",
			"",
			"PJSIP/call-out/sip:test@test.com:5060",
			"",
			"",
			"",
			&rabbitmq.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmq.Request{
				URI:      "/ari/channels/create",
				Method:   rabbitmq.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     `{"endpoint":"PJSIP/call-out/sip:test@test.com:5060","app":"voipbin","channelId":"adf2ec1a-9ee6-11ea-9d2e-33da3e3b92a3"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelCreate(tt.asterisk, tt.chanelID, tt.appArgs, tt.endpoint, tt.otherChannelID, tt.originator, tt.formats)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAstChannelDial(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	type test struct {
		name     string
		asterisk string
		chanelID string
		caller   string
		timeout  int
		response *rabbitmq.Response

		expectTarget  string
		expectRequest *rabbitmq.Request
	}

	tests := []test{
		{
			"empty caller",
			"00:11:22:33:44:55",
			"83a188ba-a060-11ea-a777-038b061dfbc3",
			"",
			30,
			&rabbitmq.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmq.Request{
				URI:      "/ari/channels/83a188ba-a060-11ea-a777-038b061dfbc3/dial",
				Method:   rabbitmq.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     `{"timeout":30}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelDial(tt.asterisk, tt.chanelID, tt.caller, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
