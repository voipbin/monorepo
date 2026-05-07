package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
)

func Test_processV1OutboundConfigsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectCustomerID uuid.UUID
		expectReq        *outboundconfig.UpdateRequest

		responseConfig *outboundconfig.OutboundConfig
		expectRes      *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:      "/v1/outbound_configs",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"test","codecs":"PCMU"}`),
			},
			expectCustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
			expectReq: func() *outboundconfig.UpdateRequest {
				n := "test"
				c := "PCMU"
				return &outboundconfig.UpdateRequest{Name: &n, Codecs: &c}
			}(),
			responseConfig: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
				Name:       "test",
				Codecs:     "PCMU",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"test","detail":"","destination_whitelist":null,"codecs":"PCMU","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				outboundConfigHandler: mockOutboundConfig,
			}

			mockOutboundConfig.EXPECT().
				Create(gomock.Any(), tt.expectCustomerID, tt.expectReq).
				Return(tt.responseConfig, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1OutboundConfigsGet(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		pageSize  uint64
		pageToken string

		expectCustomerID uuid.UUID

		responseConfigs []*outboundconfig.OutboundConfig
		expectRes       *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:    "/v1/outbound_configs?customer_id=b3fe6c84-f7f5-11ef-92b3-0be9c3b04574&page_size=10",
				Method: sock.RequestMethodGet,
			},
			pageSize:         10,
			pageToken:        "",
			expectCustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
			responseConfigs: []*outboundconfig.OutboundConfig{
				{
					ID:         uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
					CustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
					Name:       "test",
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"test","detail":"","destination_whitelist":null,"codecs":"","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				outboundConfigHandler: mockOutboundConfig,
			}

			mockOutboundConfig.EXPECT().
				List(gomock.Any(), tt.expectCustomerID, tt.pageSize, tt.pageToken).
				Return(tt.responseConfigs, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1OutboundConfigsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectID       uuid.UUID
		responseConfig *outboundconfig.OutboundConfig
		expectRes      *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:    "/v1/outbound_configs/c1234567-f7f5-11ef-92b3-0be9c3b04574",
				Method: sock.RequestMethodGet,
			},
			expectID: uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
			responseConfig: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
				Name:       "my-config",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"my-config","detail":"","destination_whitelist":null,"codecs":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				outboundConfigHandler: mockOutboundConfig,
			}

			mockOutboundConfig.EXPECT().
				GetByID(gomock.Any(), tt.expectID).
				Return(tt.responseConfig, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1OutboundConfigsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectID       uuid.UUID
		responseConfig *outboundconfig.OutboundConfig
		expectRes      *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:    "/v1/outbound_configs/c1234567-f7f5-11ef-92b3-0be9c3b04574",
				Method: sock.RequestMethodDelete,
			},
			expectID: uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
			responseConfig: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
				Name:       "to-delete",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"to-delete","detail":"","destination_whitelist":null,"codecs":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				outboundConfigHandler: mockOutboundConfig,
			}

			mockOutboundConfig.EXPECT().
				Delete(gomock.Any(), tt.expectID).
				Return(tt.responseConfig, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1OutboundConfigsIDPut(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectID       uuid.UUID
		expectReq      *outboundconfig.UpdateRequest
		responseConfig *outboundconfig.OutboundConfig
		expectRes      *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:      "/v1/outbound_configs/c1234567-f7f5-11ef-92b3-0be9c3b04574",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"updated"}`),
			},
			expectID: uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
			expectReq: func() *outboundconfig.UpdateRequest {
				n := "updated"
				return &outboundconfig.UpdateRequest{Name: &n}
			}(),
			responseConfig: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
				Name:       "updated",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"updated","detail":"","destination_whitelist":null,"codecs":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "all fields populated",
			request: &sock.Request{
				URI:      "/v1/outbound_configs/c1234567-f7f5-11ef-92b3-0be9c3b04574",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"my-config","detail":"detail-text","destination_whitelist":["us","kr"],"codecs":"PCMU,PCMA"}`),
			},
			expectID: uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
			expectReq: func() *outboundconfig.UpdateRequest {
				n := "my-config"
				d := "detail-text"
				w := []string{"us", "kr"}
				c := "PCMU,PCMA"
				return &outboundconfig.UpdateRequest{Name: &n, Detail: &d, DestinationWhitelist: &w, Codecs: &c}
			}(),
			responseConfig: &outboundconfig.OutboundConfig{
				ID:                   uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID:           uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
				Name:                 "my-config",
				Detail:               "detail-text",
				DestinationWhitelist: []string{"us", "kr"},
				Codecs:               "PCMU,PCMA",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"my-config","detail":"detail-text","destination_whitelist":["us","kr"],"codecs":"PCMU,PCMA","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				outboundConfigHandler: mockOutboundConfig,
			}

			mockOutboundConfig.EXPECT().
				Update(gomock.Any(), tt.expectID, tt.expectReq).
				Return(tt.responseConfig, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
