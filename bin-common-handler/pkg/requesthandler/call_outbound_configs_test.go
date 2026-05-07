package requesthandler

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CallV1OutboundConfigCreate(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		req        *cmoutboundconfig.UpdateRequest

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cmoutboundconfig.OutboundConfig
	}{
		{
			name: "all fields populated",

			customerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
			req: func() *cmoutboundconfig.UpdateRequest {
				n := "my-config"
				d := "detail-text"
				w := []string{"us", "kr"}
				c := "PCMU,PCMA"
				return &cmoutboundconfig.UpdateRequest{Name: &n, Detail: &d, DestinationWhitelist: &w, Codecs: &c}
			}(),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/outbound_configs",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"my-config","detail":"detail-text","destination_whitelist":["us","kr"],"codecs":"PCMU,PCMA"}`),
			},
			expectRes: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
			},
		},
		{
			name: "only name set",

			customerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
			req: func() *cmoutboundconfig.UpdateRequest {
				n := "minimal"
				return &cmoutboundconfig.UpdateRequest{Name: &n}
			}(),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/outbound_configs",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"minimal"}`),
			},
			expectRes: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID: uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
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

			// Wire-format guard: marshaled body must NOT contain a "request" wrapper key.
			if strings.Contains(string(tt.expectRequest.Data), `"request":`) {
				t.Fatalf("expected flat wire format, but expectRequest.Data contains a `request` wrapper: %s", tt.expectRequest.Data)
			}
			// Round-trip guard: marshaling the input UpdateRequest must yield top-level fields, no wrapper.
			marshaled, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("could not marshal req: %v", err)
			}
			if strings.Contains(string(marshaled), `"request":`) {
				t.Fatalf("UpdateRequest unexpectedly serialized with a `request` wrapper: %s", marshaled)
			}

			res, err := reqHandler.CallV1OutboundConfigCreate(ctx, tt.customerID, tt.req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(*tt.expectRes, *res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_CallV1OutboundConfigUpdate(t *testing.T) {
	tests := []struct {
		name string

		id  uuid.UUID
		req *cmoutboundconfig.UpdateRequest

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cmoutboundconfig.OutboundConfig
	}{
		{
			name: "all fields populated",

			id: uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
			req: func() *cmoutboundconfig.UpdateRequest {
				n := "my-config"
				d := "detail-text"
				w := []string{"us", "kr"}
				c := "PCMU,PCMA"
				return &cmoutboundconfig.UpdateRequest{Name: &n, Detail: &d, DestinationWhitelist: &w, Codecs: &c}
			}(),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","customer_id":"b3fe6c84-f7f5-11ef-92b3-0be9c3b04574","name":"my-config","detail":"detail-text","destination_whitelist":["us","kr"],"codecs":"PCMU,PCMA"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/outbound_configs/c1234567-f7f5-11ef-92b3-0be9c3b04574",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"my-config","detail":"detail-text","destination_whitelist":["us","kr"],"codecs":"PCMU,PCMA"}`),
			},
			expectRes: &cmoutboundconfig.OutboundConfig{
				ID:                   uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				CustomerID:           uuid.FromStringOrNil("b3fe6c84-f7f5-11ef-92b3-0be9c3b04574"),
				Name:                 "my-config",
				Detail:               "detail-text",
				DestinationWhitelist: []string{"us", "kr"},
				Codecs:               "PCMU,PCMA",
			},
		},
		{
			name: "only name set",

			id: uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
			req: func() *cmoutboundconfig.UpdateRequest {
				n := "renamed"
				return &cmoutboundconfig.UpdateRequest{Name: &n}
			}(),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1234567-f7f5-11ef-92b3-0be9c3b04574","name":"renamed"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/outbound_configs/c1234567-f7f5-11ef-92b3-0be9c3b04574",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"renamed"}`),
			},
			expectRes: &cmoutboundconfig.OutboundConfig{
				ID:   uuid.FromStringOrNil("c1234567-f7f5-11ef-92b3-0be9c3b04574"),
				Name: "renamed",
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

			// Wire-format guard: marshaled body must NOT contain a "request" wrapper key.
			if strings.Contains(string(tt.expectRequest.Data), `"request":`) {
				t.Fatalf("expected flat wire format, but expectRequest.Data contains a `request` wrapper: %s", tt.expectRequest.Data)
			}

			res, err := reqHandler.CallV1OutboundConfigUpdate(ctx, tt.id, tt.req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(*tt.expectRes, *res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}
