package listenhandler

import (
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

func TestNewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
	mockChannel := channelhandler.NewMockChannelHandler(mc)
	mockRecording := recordinghandler.NewMockRecordingHandler(mc)
	mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
	mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

	h := NewListenHandler(
		mockSock,
		mockCall,
		mockConfbridge,
		mockChannel,
		mockRecording,
		mockExternalMedia,
		mockGroupcall,
	)

	if h == nil {
		t.Error("NewListenHandler returned nil")
	}
}

func Test_simpleResponse(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected *sock.Response
	}{
		{
			name: "200 OK",
			code: 200,
			expected: &sock.Response{
				StatusCode: 200,
			},
		},
		{
			name: "400 Bad Request",
			code: 400,
			expected: &sock.Response{
				StatusCode: 400,
			},
		},
		{
			name: "404 Not Found",
			code: 404,
			expected: &sock.Response{
				StatusCode: 404,
			},
		},
		{
			name: "500 Internal Server Error",
			code: 500,
			expected: &sock.Response{
				StatusCode: 500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := simpleResponse(tt.code)
			if !reflect.DeepEqual(res, tt.expected) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expected, res)
			}
		})
	}
}

func Test_processRequest_notFound(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			name: "unknown URI returns 404",
			request: &sock.Request{
				URI:    "/v1/unknown/endpoint",
				Method: sock.RequestMethodGet,
			},
			expectRes: &sock.Response{
				StatusCode: 404,
			},
		},
		{
			name: "unknown method on valid URI returns 404",
			request: &sock.Request{
				URI:    "/v1/calls",
				Method: sock.RequestMethodGet, // GET /v1/calls without query string doesn't match
			},
			expectRes: &sock.Response{
				StatusCode: 404,
			},
		},
		{
			name: "empty URI returns 404",
			request: &sock.Request{
				URI:    "",
				Method: sock.RequestMethodGet,
			},
			expectRes: &sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processRequest_routingCalls(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectCode int
	}{
		{
			name: "GET /v1/calls routes correctly",
			request: &sock.Request{
				URI:    "/v1/calls?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{}`),
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockCall := h.callHandler.(*callhandler.MockCallHandler)
				mockCall.EXPECT().List(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*call.Call{}, nil)
			},
			expectCode: 200,
		},
		{
			name: "GET /v1/calls/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/calls/638769c2-620d-11eb-bd1f-6b576e26b4e6",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockCall := h.callHandler.(*callhandler.MockCallHandler)
				mockCall.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6")).Return(&call.Call{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6"),
					},
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "DELETE /v1/calls/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/calls/638769c2-620d-11eb-bd1f-6b576e26b4e6",
				Method: sock.RequestMethodDelete,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockCall := h.callHandler.(*callhandler.MockCallHandler)
				mockCall.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6")).Return(&call.Call{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6"),
					},
				}, nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expected: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_routingConfbridges(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectCode int
	}{
		{
			name: "POST /v1/confbridges routes correctly",
			request: &sock.Request{
				URI:    "/v1/confbridges",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id":"a09c9c80-98f5-11ed-a7d4-eb729c335ae0"}`),
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockConfbridge := h.confbridgeHandler.(*confbridgehandler.MockConfbridgeHandler)
				mockConfbridge.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&confbridge.Confbridge{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
					},
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "GET /v1/confbridges/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/confbridges/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockConfbridge := h.confbridgeHandler.(*confbridgehandler.MockConfbridgeHandler)
				mockConfbridge.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&confbridge.Confbridge{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
					},
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "DELETE /v1/confbridges/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/confbridges/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodDelete,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockConfbridge := h.confbridgeHandler.(*confbridgehandler.MockConfbridgeHandler)
				mockConfbridge.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&confbridge.Confbridge{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
					},
				}, nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expected: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_routingExternalMedias(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectCode int
	}{
		{
			name: "GET /v1/external-medias routes correctly",
			request: &sock.Request{
				URI:    "/v1/external-medias?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{}`),
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockExternalMedia := h.externalMediaHandler.(*externalmediahandler.MockExternalMediaHandler)
				mockExternalMedia.EXPECT().List(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*externalmedia.ExternalMedia{}, nil)
			},
			expectCode: 200,
		},
		{
			name: "GET /v1/external-medias/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/external-medias/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockExternalMedia := h.externalMediaHandler.(*externalmediahandler.MockExternalMediaHandler)
				mockExternalMedia.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&externalmedia.ExternalMedia{
					ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "DELETE /v1/external-medias/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/external-medias/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodDelete,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockExternalMedia := h.externalMediaHandler.(*externalmediahandler.MockExternalMediaHandler)
				mockExternalMedia.EXPECT().Stop(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&externalmedia.ExternalMedia{
					ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
				}, nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expected: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_routingGroupcalls(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectCode int
	}{
		{
			name: "GET /v1/groupcalls routes correctly",
			request: &sock.Request{
				URI:    "/v1/groupcalls?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{}`),
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockGroupcall := h.groupcallHandler.(*groupcallhandler.MockGroupcallHandler)
				mockGroupcall.EXPECT().List(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*groupcall.Groupcall{}, nil)
			},
			expectCode: 200,
		},
		{
			name: "GET /v1/groupcalls/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/groupcalls/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockGroupcall := h.groupcallHandler.(*groupcallhandler.MockGroupcallHandler)
				mockGroupcall.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&groupcall.Groupcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
					},
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "DELETE /v1/groupcalls/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/groupcalls/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodDelete,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockGroupcall := h.groupcallHandler.(*groupcallhandler.MockGroupcallHandler)
				mockGroupcall.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&groupcall.Groupcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
					},
				}, nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expected: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_routingRecordings(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectCode int
	}{
		{
			name: "GET /v1/recordings routes correctly",
			request: &sock.Request{
				URI:    "/v1/recordings?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{}`),
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockRecording := h.recordingHandler.(*recordinghandler.MockRecordingHandler)
				mockRecording.EXPECT().List(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*recording.Recording{}, nil)
			},
			expectCode: 200,
		},
		{
			name: "GET /v1/recordings/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/recordings/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockRecording := h.recordingHandler.(*recordinghandler.MockRecordingHandler)
				mockRecording.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&recording.Recording{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
					},
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "DELETE /v1/recordings/<id> routes correctly",
			request: &sock.Request{
				URI:    "/v1/recordings/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodDelete,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockRecording := h.recordingHandler.(*recordinghandler.MockRecordingHandler)
				mockRecording.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(&recording.Recording{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
					},
				}, nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expected: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_routingRecovery(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectCode int
	}{
		{
			name: "POST /v1/recovery routes correctly",
			request: &sock.Request{
				URI:    "/v1/recovery",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"asterisk_id":"test-asterisk"}`),
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockCall := h.callHandler.(*callhandler.MockCallHandler)
				mockCall.EXPECT().RecoveryStart(gomock.Any(), "test-asterisk").Return(nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expected: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_routingChannels(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectNil  bool
	}{
		{
			name: "POST /v1/channels/<id>/health-check routes correctly",
			request: &sock.Request{
				URI:    "/v1/channels/test-channel-123/health-check",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"retry_count": 0}`),
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockChannel := h.channelHandler.(*channelhandler.MockChannelHandler)
				mockChannel.EXPECT().HealthCheck(gomock.Any(), "test-channel-123", 0)
			},
			expectNil: true, // This handler returns nil response
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectNil {
				if res != nil {
					t.Errorf("Expected nil response, got: %v", res)
				}
			}
		})
	}
}

func Test_processRequest_errorPaths(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		setupMocks func(mc *gomock.Controller, h *listenHandler)
		expectCode int
	}{
		{
			name: "GET /v1/calls/<id> returns 404 on error",
			request: &sock.Request{
				URI:    "/v1/calls/638769c2-620d-11eb-bd1f-6b576e26b4e6",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockCall := h.callHandler.(*callhandler.MockCallHandler)
				mockCall.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6")).Return(nil, fmt.Errorf("not found"))
			},
			expectCode: 404,
		},
		{
			name: "GET /v1/confbridges/<id> returns 400 on error",
			request: &sock.Request{
				URI:    "/v1/confbridges/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockConfbridge := h.confbridgeHandler.(*confbridgehandler.MockConfbridgeHandler)
				mockConfbridge.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(nil, fmt.Errorf("not found"))
			},
			expectCode: 400,
		},
		{
			name: "GET /v1/recordings/<id> returns 404 on error",
			request: &sock.Request{
				URI:    "/v1/recordings/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockRecording := h.recordingHandler.(*recordinghandler.MockRecordingHandler)
				mockRecording.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(nil, fmt.Errorf("not found"))
			},
			expectCode: 404,
		},
		{
			name: "GET /v1/external-medias/<id> returns 404 on error",
			request: &sock.Request{
				URI:    "/v1/external-medias/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockExternalMedia := h.externalMediaHandler.(*externalmediahandler.MockExternalMediaHandler)
				mockExternalMedia.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(nil, fmt.Errorf("not found"))
			},
			expectCode: 404,
		},
		{
			name: "GET /v1/groupcalls/<id> returns 500 on error",
			request: &sock.Request{
				URI:    "/v1/groupcalls/68e9edd8-3609-11ec-ad76-b72fa8f57f23",
				Method: sock.RequestMethodGet,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockGroupcall := h.groupcallHandler.(*groupcallhandler.MockGroupcallHandler)
				mockGroupcall.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23")).Return(nil, fmt.Errorf("not found"))
			},
			expectCode: 500,
		},
		{
			name: "DELETE /v1/calls/<id> returns 500 on error",
			request: &sock.Request{
				URI:    "/v1/calls/638769c2-620d-11eb-bd1f-6b576e26b4e6",
				Method: sock.RequestMethodDelete,
			},
			setupMocks: func(mc *gomock.Controller, h *listenHandler) {
				mockCall := h.callHandler.(*callhandler.MockCallHandler)
				mockCall.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6")).Return(nil, fmt.Errorf("delete error"))
			},
			expectCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				confbridgeHandler:    mockConfbridge,
				channelHandler:       mockChannel,
				recordingHandler:     mockRecording,
				externalMediaHandler: mockExternalMedia,
				groupcallHandler:     mockGroupcall,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mc, h)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expected: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}
