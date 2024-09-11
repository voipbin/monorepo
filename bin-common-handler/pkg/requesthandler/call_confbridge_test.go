package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_CallV1ConfbridgeCreate(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		confbridgeType cmconfbridge.Type

		response      *sock.Response
		expectTarget  string
		expectRequest *sock.Request

		expectRes *cmconfbridge.Confbridge
	}

	tests := []test{
		{
			"type connect",

			uuid.FromStringOrNil("a72262a0-9978-11ed-bb1a-4745c1dde2fa"),
			cmconfbridge.TypeConnect,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"700a6ca0-5ba2-11ec-98bd-a3b749617d0b"}`),
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/confbridges",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a72262a0-9978-11ed-bb1a-4745c1dde2fa","type":"connect"}`),
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("700a6ca0-5ba2-11ec-98bd-a3b749617d0b"),
			},
		},
		{
			"type conference",

			uuid.FromStringOrNil("ac15b0dc-9978-11ed-b5db-a729c9e168dd"),
			cmconfbridge.TypeConference,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a8d56354-978f-11ec-b4a0-2f9706b7c3ff"}`),
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/confbridges",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ac15b0dc-9978-11ed-b5db-a729c9e168dd","type":"conference"}`),
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("a8d56354-978f-11ec-b4a0-2f9706b7c3ff"),
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
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeCreate(ctx, tt.customerID, tt.confbridgeType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeGet(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("97e26732-9049-11ed-ac5c-871c69c4583b"),

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/confbridges/97e26732-9049-11ed-ac5c-871c69c4583b",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"97e26732-9049-11ed-ac5c-871c69c4583b"}`),
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("97e26732-9049-11ed-ac5c-871c69c4583b"),
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
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeGet(ctx, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeExternalMediaStart(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID   uuid.UUID
		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("8bb7a268-97d0-11ed-bb1d-efd9a3f33560"),
			"localhost:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8bb7a268-97d0-11ed-bb1d-efd9a3f33560"}`),
			},

			&sock.Request{
				URI:      "/v1/confbridges/8bb7a268-97d0-11ed-bb1d-efd9a3f33560/external-media",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"external_host":"localhost:5060","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("8bb7a268-97d0-11ed-bb1d-efd9a3f33560"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeExternalMediaStart(ctx, tt.confbridgeID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeExternalMediaStop(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("8c21d002-97d0-11ed-9bb5-bf7c25553a09"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8c21d002-97d0-11ed-9bb5-bf7c25553a09"}`),
			},

			&sock.Request{
				URI:    "/v1/confbridges/8c21d002-97d0-11ed-9bb5-bf7c25553a09/external-media",
				Method: sock.RequestMethodDelete,
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("8c21d002-97d0-11ed-9bb5-bf7c25553a09"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeExternalMediaStop(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeRecordingStart(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID
		format       cmrecording.Format
		endOfSilence int
		endOfKey     string
		duration     int

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("9ab869b4-9979-11ed-ae1a-1fd050fd5c80"),
			cmrecording.FormatWAV,
			1000,
			"#",
			86400,

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"9ab869b4-9979-11ed-ae1a-1fd050fd5c80"}`),
			},

			&sock.Request{
				URI:      "/v1/confbridges/9ab869b4-9979-11ed-ae1a-1fd050fd5c80/recording_start",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"format":"wav","end_of_silence":1000,"end_of_key":"#","duration":86400}`),
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("9ab869b4-9979-11ed-ae1a-1fd050fd5c80"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeRecordingStart(ctx, tt.confbridgeID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeRecordingStop(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("9aeabe1e-9979-11ed-9bde-bbb0da66dc29"),

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"9aeabe1e-9979-11ed-9bde-bbb0da66dc29"}`),
			},

			&sock.Request{
				URI:    "/v1/confbridges/9aeabe1e-9979-11ed-9bde-bbb0da66dc29/recording_stop",
				Method: sock.RequestMethodPost,
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("9aeabe1e-9979-11ed-9bde-bbb0da66dc29"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeRecordingStop(ctx, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeFlagAdd(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID
		flag         cmconfbridge.Flag

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("367d97a2-d7be-11ed-90bb-3354b92cec8a"),
			cmconfbridge.FlagNoAutoLeave,

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"367d97a2-d7be-11ed-90bb-3354b92cec8a"}`),
			},

			&sock.Request{
				URI:      "/v1/confbridges/367d97a2-d7be-11ed-90bb-3354b92cec8a/flags",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"flag":"no_auto_leave"}`),
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("367d97a2-d7be-11ed-90bb-3354b92cec8a"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeFlagAdd(ctx, tt.confbridgeID, tt.flag)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeFlagRemove(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID
		flag         cmconfbridge.Flag

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("96c67bf6-d7be-11ed-abd3-efffaa03c246"),
			cmconfbridge.FlagNoAutoLeave,

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"96c67bf6-d7be-11ed-abd3-efffaa03c246"}`),
			},

			&sock.Request{
				URI:      "/v1/confbridges/96c67bf6-d7be-11ed-abd3-efffaa03c246/flags",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"flag":"no_auto_leave"}`),
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("96c67bf6-d7be-11ed-abd3-efffaa03c246"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeFlagRemove(ctx, tt.confbridgeID, tt.flag)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeTerminate(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("96d39178-dae9-11ed-92a2-17288622d986"),

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"96d39178-dae9-11ed-92a2-17288622d986"}`),
			},

			&sock.Request{
				URI:    "/v1/confbridges/96d39178-dae9-11ed-92a2-17288622d986/terminate",
				Method: sock.RequestMethodPost,
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("96d39178-dae9-11ed-92a2-17288622d986"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ConfbridgeTerminate(ctx, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ConfbridgeRing(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("2d37d7f0-db8f-11ed-a1f7-53a2bd6a697d"),

			&sock.Response{
				StatusCode: 200,
			},

			&sock.Request{
				URI:    "/v1/confbridges/2d37d7f0-db8f-11ed-a1f7-53a2bd6a697d/ring",
				Method: sock.RequestMethodPost,
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("2d37d7f0-db8f-11ed-a1f7-53a2bd6a697d"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			if errRing := reqHandler.CallV1ConfbridgeRing(ctx, tt.confbridgeID); errRing != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRing)
			}
		})
	}
}

func Test_CallV1ConfbridgeAnswer(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmconfbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("2d7289f4-db8f-11ed-8c13-efbe206011b3"),

			&sock.Response{
				StatusCode: 200,
			},

			&sock.Request{
				URI:    "/v1/confbridges/2d7289f4-db8f-11ed-8c13-efbe206011b3/answer",
				Method: sock.RequestMethodPost,
			},
			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("2d7289f4-db8f-11ed-8c13-efbe206011b3"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			if errRing := reqHandler.CallV1ConfbridgeAnswer(ctx, tt.confbridgeID); errRing != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRing)
			}
		})
	}
}
