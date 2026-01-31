package requesthandler

import (
	"context"
	"reflect"
	"testing"

	amtool "monorepo/bin-ai-manager/models/tool"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AIV1ToolList(t *testing.T) {

	tests := []struct {
		name string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []amtool.Tool
	}{
		{
			name: "normal - returns all tools",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"tools":[{"name":"connect_call","description":"Connects to another endpoint","parameters":{"type":"object"}},{"name":"send_email","description":"Sends an email","parameters":{"type":"object"}}]}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/tools",
				Method: sock.RequestMethodGet,
			},
			expectRes: []amtool.Tool{
				{
					Name:        amtool.ToolNameConnectCall,
					Description: "Connects to another endpoint",
					Parameters:  map[string]any{"type": "object"},
				},
				{
					Name:        amtool.ToolNameSendEmail,
					Description: "Sends an email",
					Parameters:  map[string]any{"type": "object"},
				},
			},
		},
		{
			name: "empty tools",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"tools":[]}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/tools",
				Method: sock.RequestMethodGet,
			},
			expectRes: []amtool.Tool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.AIV1ToolList(ctx)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
