package requesthandler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_AstRecordingStop(t *testing.T) {

	tests := []struct {
		name        string
		asteriskID  string
		recordingID string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmqhandler.RequestMethod
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"74b441de-90be-11ed-a5ab-eff9d8e46ebe",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/recordings/live/74b441de-90be-11ed-a5ab-eff9d8e46ebe/stop",
			rabbitmqhandler.RequestMethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

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

			err := reqHandler.AstRecordingStop(context.Background(), tt.asteriskID, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstRecordingPause(t *testing.T) {

	tests := []struct {
		name        string
		asteriskID  string
		recordingID string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmqhandler.RequestMethod
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"ac61de34-90be-11ed-9fd7-b3becbae66ed",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/recordings/live/ac61de34-90be-11ed-9fd7-b3becbae66ed/pause",
			rabbitmqhandler.RequestMethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

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

			err := reqHandler.AstRecordingPause(context.Background(), tt.asteriskID, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstRecordingUnpause(t *testing.T) {

	tests := []struct {
		name        string
		asteriskID  string
		recordingID string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmqhandler.RequestMethod
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"ac61de34-90be-11ed-9fd7-b3becbae66ed",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/recordings/live/ac61de34-90be-11ed-9fd7-b3becbae66ed/pause",
			rabbitmqhandler.RequestMethodDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

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

			err := reqHandler.AstRecordingUnpause(context.Background(), tt.asteriskID, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstRecordingMute(t *testing.T) {

	tests := []struct {
		name        string
		asteriskID  string
		recordingID string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmqhandler.RequestMethod
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"e73398fe-90be-11ed-821f-0fa720b0f3ab",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/recordings/live/e73398fe-90be-11ed-821f-0fa720b0f3ab/mute",
			rabbitmqhandler.RequestMethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

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

			err := reqHandler.AstRecordingMute(context.Background(), tt.asteriskID, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func Test_AstRecordingUnmute(t *testing.T) {

	tests := []struct {
		name        string
		asteriskID  string
		recordingID string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmqhandler.RequestMethod
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"f8feed4a-90be-11ed-aa76-c7eb32286c6d",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/recordings/live/f8feed4a-90be-11ed-aa76-c7eb32286c6d/mute",
			rabbitmqhandler.RequestMethodDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

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

			err := reqHandler.AstRecordingUnmute(context.Background(), tt.asteriskID, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}
