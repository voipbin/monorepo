package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmbridge "monorepo/bin-call-manager/models/bridge"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_AstBridgeCreate(t *testing.T) {

	tests := []struct {
		name        string
		asteriskID  string
		bridgeID    string
		bridgeName  string
		bridgeTypes []cmbridge.Type

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			name:        "normal",
			asteriskID:  "00:11:22:33:44:55",
			bridgeID:    "5f573260-549f-11ee-8c9c-a33cb00ec17b",
			bridgeName:  "reference_type=call,reference_id=67ab1e68-549f-11ee-bab0-575214e7ccd7",
			bridgeTypes: []cmbridge.Type{cmbridge.TypeMixing, cmbridge.TypeProxyMedia},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"5f573260-549f-11ee-8c9c-a33cb00ec17b"}`),
			},

			expectTarget: "asterisk.00:11:22:33:44:55.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/ari/bridges",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"type":"mixing,proxy_media","bridgeId":"5f573260-549f-11ee-8c9c-a33cb00ec17b","name":"reference_type=call,reference_id=67ab1e68-549f-11ee-bab0-575214e7ccd7"}`),
			},
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			if err := reqHandler.AstBridgeCreate(context.Background(), tt.asteriskID, tt.bridgeID, tt.bridgeName, tt.bridgeTypes); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AstBridgeDelete(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		bridgeID   string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			name:       "normal",
			asteriskID: "00:11:22:33:44:55",
			bridgeID:   "8d815688-54a0-11ee-bc4f-6fd312bf1408",
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"8d815688-54a0-11ee-bc4f-6fd312bf1408"}`),
			},

			expectTarget: "asterisk.00:11:22:33:44:55.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/ari/bridges/8d815688-54a0-11ee-bc4f-6fd312bf1408",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			if err := reqHandler.AstBridgeDelete(context.Background(), tt.asteriskID, tt.bridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AstBridgeGet(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		bridgeID   string
		response   *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectBridge  *cmbridge.Bridge
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"261a2496-dc28-11ea-b3b2-afa07bdffeb2",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"3e6eec96-fabe-4041-870d-e1daee11aafb","technology":"softmix","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"reference_type=confbridge,reference_id=60d7ee79-78f5-4c86-9d34-4c699e8d5ee7","channels":[],"creationtime":"2020-08-10T22:50:28.085+0000","video_mode":"sfu"}`),
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/bridges/261a2496-dc28-11ea-b3b2-afa07bdffeb2",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     nil,
			},
			&cmbridge.Bridge{
				ID:   "3e6eec96-fabe-4041-870d-e1daee11aafb",
				Name: "reference_type=confbridge,reference_id=60d7ee79-78f5-4c86-9d34-4c699e8d5ee7",

				// info
				Type:    cmbridge.TypeMixing,
				Tech:    cmbridge.TechSoftmix,
				Class:   "stasis",
				Creator: "Stasis",

				VideoMode: "sfu",
				// VideoSourceID: "",

				ChannelIDs: []string{},

				// conference info
				ReferenceType: cmbridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("60d7ee79-78f5-4c86-9d34-4c699e8d5ee7"),

				TMCreate: "",
				TMUpdate: "",
				TMDelete: "",
			},
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			res, err := reqHandler.AstBridgeGet(context.Background(), tt.asteriskID, tt.bridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectBridge, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectBridge, res)
			}

		})
	}
}

func Test_AstBridgeAddChannel(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		bridgeID   string
		channelID  string
		role       string
		absorbDTMF bool
		mute       bool
		response   *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			name:       "normal",
			asteriskID: "00:11:22:33:44:55",
			bridgeID:   "4175719c-54a1-11ee-89d8-d3ee36ac6a81",
			channelID:  "41a2da42-54a1-11ee-84ac-7b1cf34a10d3",
			role:       "",
			absorbDTMF: true,
			mute:       true,

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
			},

			expectTarget: "asterisk.00:11:22:33:44:55.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/ari/bridges/4175719c-54a1-11ee-89d8-d3ee36ac6a81/addChannel",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"channel":"41a2da42-54a1-11ee-84ac-7b1cf34a10d3","absorbDTMF":true,"mute":true}`),
			},
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			if err := reqHandler.AstBridgeAddChannel(context.Background(), tt.asteriskID, tt.bridgeID, tt.channelID, tt.role, tt.absorbDTMF, tt.mute); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AstBridgeRemoveChannel(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		bridgeID   string
		channelID  string
		response   *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			name:       "normal",
			asteriskID: "00:11:22:33:44:55",
			bridgeID:   "bd6b2914-54a0-11ee-a509-a725e1be2974",
			channelID:  "bd928b58-54a0-11ee-8831-ef3bd4ff798f",

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
			},

			expectTarget: "asterisk.00:11:22:33:44:55.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/ari/bridges/bd6b2914-54a0-11ee-a509-a725e1be2974/removeChannel",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"channel":"bd928b58-54a0-11ee-8831-ef3bd4ff798f"}`),
			},
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			if err := reqHandler.AstBridgeRemoveChannel(context.Background(), tt.asteriskID, tt.bridgeID, tt.channelID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AstBridgeRecord(t *testing.T) {

	tests := []struct {
		name     string
		asterisk string
		bridgeID string
		filename string
		format   string
		duration int
		silence  int
		beep     bool
		endKey   string
		ifExist  string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"67708fbc-904d-11ed-beba-4f35dd737a8d",
			"conference_67708fbc-904d-11ed-beba-4f35dd737a8d_2020-05-17T10:24:54.396+0000",
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
				URI:      "/ari/bridges/67708fbc-904d-11ed-beba-4f35dd737a8d/record",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"conference_67708fbc-904d-11ed-beba-4f35dd737a8d_2020-05-17T10:24:54.396+0000","format":"wav","maxDurationSeconds":0,"maxSilenceSeconds":0,"beep":false,"terminateOn":"","ifExists":"fail"}`),
			},
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstBridgeRecord(context.Background(), tt.asterisk, tt.bridgeID, tt.filename, tt.format, tt.duration, tt.silence, tt.beep, tt.endKey, tt.ifExist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
