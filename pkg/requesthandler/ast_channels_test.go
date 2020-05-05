package requesthandler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

func TestSetSock(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	if mockSock == nil {
		t.Errorf("Error")
	}

	reqHandler := NewRequestHandler(mockSock)

	if reqHandler == nil {
		t.Errorf("Wrong match. expact: true, got: false")
	}
}

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

			"asterisk_ari_request-00:11:22:33:44:55",
			"/ari/channels/5734c890-7f6e-11ea-9520-6f774800cd74/answer",
			rabbitmq.RequestMethodPost,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock)

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
			"asterisk_ari_request-00:11:22:33:44:55",
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
			"asterisk_ari_request-00:11:22:33:44:55",
			rabbitmq.RequestMethodPost,
			`{"context":"test-context","extension":"testcall","priority":1,"label":""}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock)

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
			"asterisk_ari_request-00:11:22:33:44:55",
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
			"asterisk_ari_request-00:11:22:33:44:55",
			rabbitmq.RequestMethodPost,
			`{"variable":"test-variable","value":""}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock)

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
			"asterisk_ari_request-00:11:22:33:44:55",
			rabbitmq.RequestMethodDelete,
			`{"reason_code":"16"}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock)

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

	type test struct {
		name       string
		asteriskID string
		channelID  string
		snoopID    string
		appArgs    string
		spy        ChannelSnoopDirection
		whisper    ChannelSnoopDirection

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
			ChannelSnoopDirectionIn,
			ChannelSnoopDirectionIn,

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk_ari_request-00:11:22:33:44:55",
			rabbitmq.RequestMethodPost,
			`{"spy":"in","whisper":"in","app":"voipbin","appArgs":"test","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`,
		},
		{
			"whisper is none",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"",
			ChannelSnoopDirectionIn,
			ChannelSnoopDirectionNone,

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk_ari_request-00:11:22:33:44:55",
			rabbitmq.RequestMethodPost,
			`{"spy":"in","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`,
		},
		{
			"Spy is none",
			"00:11:22:33:44:55",
			"a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d",
			"acc09eea-8dd0-11ea-99ba-e311d0dcd408",
			"",
			ChannelSnoopDirectionNone,
			ChannelSnoopDirectionBoth,

			"/ari/channels/a7d0241e-8dd0-11ea-9b06-7b0ced5bf93d/snoop",
			"asterisk_ari_request-00:11:22:33:44:55",
			rabbitmq.RequestMethodPost,
			`{"whisper":"both","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock)

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
