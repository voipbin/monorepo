package listenhandler

import (
	"encoding/json"
	"reflect"
	"testing"

	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/response"
	"monorepo/bin-ai-manager/pkg/toolhandler"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

func Test_processV1ToolsGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseTools []tool.Tool

		expectStatusCode int
	}{
		{
			name: "normal - returns all tools",
			request: &sock.Request{
				URI:    "/v1/tools",
				Method: sock.RequestMethodGet,
			},
			responseTools: []tool.Tool{
				{
					Name:        tool.ToolNameConnectCall,
					Description: "Connects to another endpoint",
					Parameters:  map[string]any{"type": "object"},
				},
				{
					Name:        tool.ToolNameSendEmail,
					Description: "Sends an email",
					Parameters:  map[string]any{"type": "object"},
				},
			},
			expectStatusCode: 200,
		},
		{
			name: "empty tools",
			request: &sock.Request{
				URI:    "/v1/tools",
				Method: sock.RequestMethodGet,
			},
			responseTools:    []tool.Tool{},
			expectStatusCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTool := toolhandler.NewMockToolHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				toolHandler: mockTool,
			}

			mockTool.EXPECT().GetAll().Return(tt.responseTools)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("processRequest() error = %v", err)
				return
			}

			if res.StatusCode != tt.expectStatusCode {
				t.Errorf("processRequest() StatusCode = %v, want %v", res.StatusCode, tt.expectStatusCode)
				return
			}

			// Parse response and verify tools
			var gotResponse response.V1ToolsGet
			if err := json.Unmarshal(res.Data, &gotResponse); err != nil {
				t.Errorf("Could not unmarshal response: %v", err)
				return
			}

			if len(gotResponse.Tools) != len(tt.responseTools) {
				t.Errorf("Response has %d tools, want %d", len(gotResponse.Tools), len(tt.responseTools))
				return
			}

			for i, wantTool := range tt.responseTools {
				if gotResponse.Tools[i].Name != wantTool.Name {
					t.Errorf("Tool[%d].Name = %v, want %v", i, gotResponse.Tools[i].Name, wantTool.Name)
				}
				if gotResponse.Tools[i].Description != wantTool.Description {
					t.Errorf("Tool[%d].Description = %v, want %v", i, gotResponse.Tools[i].Description, wantTool.Description)
				}
				if !reflect.DeepEqual(gotResponse.Tools[i].Parameters, wantTool.Parameters) {
					t.Errorf("Tool[%d].Parameters = %v, want %v", i, gotResponse.Tools[i].Parameters, wantTool.Parameters)
				}
			}
		})
	}
}
