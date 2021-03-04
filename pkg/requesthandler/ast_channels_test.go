package requesthandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestAstChannelAnswer(t *testing.T) {

	type test struct {
		name       string
		asteriskID string
		channelID  string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmqhandler.RequestMethod
	}

	tests := []test{
		{
			"normal",
			"00:11:22:33:44:55",
			"5734c890-7f6e-11ea-9520-6f774800cd74",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/channels/5734c890-7f6e-11ea-9520-6f774800cd74/answer",
			rabbitmqhandler.RequestMethodPost,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmqhandler.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: "",
					Data:     nil,
				},
			).Return(&rabbitmqhandler.Response{StatusCode: 200, Data: nil}, nil)

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
		expectMethod rabbitmqhandler.RequestMethod
		expectData   []byte
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
			rabbitmqhandler.RequestMethodPost,
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
			rabbitmqhandler.RequestMethodPost,
			[]byte(`{"context":"test-context","extension":"testcall","priority":1,"label":""}`),
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmqhandler.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmqhandler.Response{StatusCode: 200, Data: nil}, nil)

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
		expectMethod rabbitmqhandler.RequestMethod
		expectData   []byte
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
			rabbitmqhandler.RequestMethodPost,

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
			rabbitmqhandler.RequestMethodPost,
			[]byte(`{"variable":"test-variable","value":""}`),
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmqhandler.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmqhandler.Response{StatusCode: 200, Data: nil}, nil)

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
		expectMethod rabbitmqhandler.RequestMethod
		expectData   []byte
	}

	tests := []test{
		{
			"have all item",
			"00:11:22:33:44:55",
			"ef6ed35e-828d-11ea-9cd9-83d7b7314faa",
			ari.ChannelCauseNormalClearing,

			"/ari/channels/ef6ed35e-828d-11ea-9cd9-83d7b7314faa",
			"asterisk.00:11:22:33:44:55.request",
			rabbitmqhandler.RequestMethodDelete,
			[]byte(`{"reason_code":"16"}`),
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmqhandler.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmqhandler.Response{StatusCode: 200, Data: nil}, nil)

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

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

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
		expectMethod rabbitmqhandler.RequestMethod
		expectData   []byte
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
			rabbitmqhandler.RequestMethodPost,
			[]byte(`{"spy":"in","whisper":"in","app":"voipbin","appArgs":"test","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`),
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
			rabbitmqhandler.RequestMethodPost,
			[]byte(`{"spy":"in","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`),
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
			rabbitmqhandler.RequestMethodPost,
			[]byte(`{"whisper":"both","app":"voipbin","snoopId":"acc09eea-8dd0-11ea-99ba-e311d0dcd408"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmqhandler.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: ContentTypeJSON,
					Data:     tt.expectData,
				},
			).Return(&rabbitmqhandler.Response{StatusCode: 200, Data: nil}, nil)

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

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	type test struct {
		name     string
		asterisk string
		id       string
		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		expectURI     string
		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"normal test",
			"00:11:22:33:44:55",
			"1589711094.100",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"1589711094.100","name":"PJSIP/call-in-00000019","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"8872616","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=xt1GqgsEfG,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-17T10:24:54.396+0000","language":"en"}`),
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/1589711094.100",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     nil,
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

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	type test struct {
		name     string
		asterisk string
		id       string
		digit    string
		duration int
		before   int
		between  int
		after    int
		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/6d11e7c2-9a69-11ea-95af-eb4a15c08df1/dtmf",
				Method:   rabbitmqhandler.RequestMethodPost,
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/6d11e7c2-9a69-11ea-95af-eb4a15c08df1/dtmf",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"dtmf":"19827348","duration":100,"before":0,"between":0,"after":0}`),
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

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	type test struct {
		name           string
		asterisk       string
		channelID      string
		appArgs        string
		endpoint       string
		otherChannelID string
		originator     string
		formats        string
		variables      map[string]string
		response       *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
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
			nil,
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/create",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"endpoint":"PJSIP/call-out/sip:test@test.com:5060","app":"voipbin","channelId":"adf2ec1a-9ee6-11ea-9d2e-33da3e3b92a3"}`),
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/create",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"endpoint":"PJSIP/call-out/sip:test@test.com:5060","app":"voipbin","channelId":"0a2628dc-0853-11eb-811f-ebaf03ba1ba6","variables":{"CALLERID(all)":"+123456789"}}`),
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelCreate(tt.asterisk, tt.channelID, tt.appArgs, tt.endpoint, tt.otherChannelID, tt.originator, tt.formats, tt.variables)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAstChannelDial(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	type test struct {
		name      string
		asterisk  string
		channelID string
		caller    string
		timeout   int
		response  *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"empty caller",
			"00:11:22:33:44:55",
			"83a188ba-a060-11ea-a777-038b061dfbc3",
			"",
			30,
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/83a188ba-a060-11ea-a777-038b061dfbc3/dial",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"timeout":30}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelDial(tt.asterisk, tt.channelID, tt.caller, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAstChannelPlay(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	type test struct {
		name      string
		asterisk  string
		channelID string
		actionID  uuid.UUID
		medias    []string
		response  *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"1 media",
			"00:11:22:33:44:55",
			"94bcc2b4-e718-11ea-a8cf-e7d1a61482a8",
			uuid.FromStringOrNil("c44864cc-e7d9-11ea-923a-73e96775044d"),
			[]string{"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/94bcc2b4-e718-11ea-a8cf-e7d1a61482a8/play",
				Method:   rabbitmqhandler.RequestMethodPost,
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/94bcc2b4-e718-11ea-a8cf-e7d1a61482a8/play",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"media":["sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm-test.wav"],"playbackId":"dde1c518-e7d9-11ea-902a-2b04669d8a49"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelPlay(tt.asterisk, tt.channelID, tt.actionID, tt.medias, "")
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAstChannelRecord(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	type test struct {
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

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
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

			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/channels/b3b6ca04-28d4-11eb-a27e-ebcd6dfed523/record",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"call_b469f3cc-28d4-11eb-b29a-db389e2bf1ca_2020-05-17T10:24:54.396+0000","format":"wav","maxDurationSeconds":0,"maxSilenceSeconds":0,"beep":false,"terminateOn":"","ifExists":"fail"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstChannelRecord(tt.asterisk, tt.channelID, tt.filename, tt.format, tt.duration, tt.silence, tt.beep, tt.endKey, tt.ifExist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
