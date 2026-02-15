package listenhandler

import (
	"context"
	"encoding/json"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processV1AgentsIDPut(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data []byte

		responseAgent *agent.Agent
		expectStatus  int
	}{
		{
			name: "normal",

			uri: "/v1/agents/" + uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83").String(),
			data: []byte(`{
				"name": "updated name",
				"detail": "updated detail",
				"ring_method": "linear"
			}`),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
				},
				Name:       "updated name",
				Detail:     "updated detail",
				RingMethod: agent.RingMethodLinear,
			},
			expectStatus: 200,
		},
		{
			name: "invalid URI",

			uri:  "/v1/agents",
			data: []byte(`{}`),

			responseAgent: nil,
			expectStatus:  400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{
				agentHandler: mockAgent,
			}
			ctx := context.Background()

			if tt.expectStatus == 200 {
				mockAgent.EXPECT().UpdateBasicInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseAgent, nil)
			}

			req := &sock.Request{
				URI:  tt.uri,
				Data: tt.data,
			}

			res, err := h.processV1AgentsIDPut(ctx, req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_processV1AgentsIDPasswordPut(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data []byte

		responseAgent *agent.Agent
		expectStatus  int
	}{
		{
			name: "normal",

			uri: "/v1/agents/" + uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83").String() + "/password",
			data: []byte(`{
				"password": "newpassword"
			}`),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
				},
			},
			expectStatus: 200,
		},
		{
			name: "invalid URI",

			uri:  "/v1/agents/password",
			data: []byte(`{}`),

			responseAgent: nil,
			expectStatus:  400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{
				agentHandler: mockAgent,
			}
			ctx := context.Background()

			if tt.expectStatus == 200 {
				mockAgent.EXPECT().UpdatePassword(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseAgent, nil)
			}

			req := &sock.Request{
				URI:  tt.uri,
				Data: tt.data,
			}

			res, err := h.processV1AgentsIDPasswordPut(ctx, req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_processV1AgentsIDTagIDsPut(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data []byte

		responseAgent *agent.Agent
		expectStatus  int
	}{
		{
			name: "normal",

			uri: "/v1/agents/" + uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83").String() + "/tag_ids",
			data: []byte(`{
				"tag_ids": ["700c10b4-4b4e-11ec-959b-bb95248c693f"]
			}`),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
				},
				TagIDs: []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
			},
			expectStatus: 200,
		},
		{
			name: "invalid URI",

			uri:  "/v1/agents/tag_ids",
			data: []byte(`{}`),

			responseAgent: nil,
			expectStatus:  400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{
				agentHandler: mockAgent,
			}
			ctx := context.Background()

			if tt.expectStatus == 200 {
				mockAgent.EXPECT().UpdateTagIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseAgent, nil)
			}

			req := &sock.Request{
				URI:  tt.uri,
				Data: tt.data,
			}

			res, err := h.processV1AgentsIDTagIDsPut(ctx, req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
		})
	}
}


func Test_processV1AgentsUsernameLogin(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data []byte

		responseAgent *agent.Agent
		expectStatus  int
	}{
		{
			name: "normal",

			uri: "/v1/agents/test@voipbin.net/login",
			data: []byte(`{
				"password": "testpassword"
			}`),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username: "test@voipbin.net",
			},
			expectStatus: 200,
		},
		{
			name: "invalid URI",

			uri:  "/v1/agents/login",
			data: []byte(`{}`),

			responseAgent: nil,
			expectStatus:  400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{
				agentHandler: mockAgent,
			}
			ctx := context.Background()

			if tt.expectStatus == 200 {
				mockAgent.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseAgent, nil)
			}

			req := &sock.Request{
				URI:  tt.uri,
				Data: tt.data,
			}

			res, err := h.processV1AgentsUsernameLogin(ctx, req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_processV1AgentsIDAddressesPut(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data []byte

		responseAgent *agent.Agent
		expectStatus  int
	}{
		{
			name: "normal",

			uri: "/v1/agents/" + uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83").String() + "/addresses",
			data: []byte(`{
				"addresses": [
					{
						"type": "tel",
						"target": "+821021656521"
					}
				]
			}`),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
				},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
					},
				},
			},
			expectStatus: 200,
		},
		{
			name: "invalid URI",

			uri:  "/v1/agents/addresses",
			data: []byte(`{}`),

			responseAgent: nil,
			expectStatus:  400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{
				agentHandler: mockAgent,
			}
			ctx := context.Background()

			if tt.expectStatus == 200 {
				mockAgent.EXPECT().UpdateAddresses(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseAgent, nil)
			}

			req := &sock.Request{
				URI:  tt.uri,
				Data: tt.data,
			}

			res, err := h.processV1AgentsIDAddressesPut(ctx, req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
		})
	}
}


func Test_processV1AgentsPost_unmarshal_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAgent := agenthandler.NewMockAgentHandler(mc)
	h := &listenHandler{
		agentHandler: mockAgent,
	}
	ctx := context.Background()

	req := &sock.Request{
		URI:  "/v1/agents",
		Data: []byte(`invalid json`),
	}

	res, err := h.processV1AgentsPost(ctx, req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong status code. expect: 400, got: %v", res.StatusCode)
	}
}

func Test_processV1AgentsIDGet_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAgent := agenthandler.NewMockAgentHandler(mc)
	h := &listenHandler{
		agentHandler: mockAgent,
	}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83")
	responseAgent := &agent.Agent{
		Identity: commonidentity.Identity{
			ID: agentID,
		},
	}

	mockAgent.EXPECT().Get(gomock.Any(), agentID).Return(responseAgent, nil)

	req := &sock.Request{
		URI: "/v1/agents/" + agentID.String(),
	}

	res, err := h.processV1AgentsIDGet(ctx, req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Wrong status code. expect: 200, got: %v", res.StatusCode)
	}

	var resAgent agent.Agent
	if err := json.Unmarshal(res.Data, &resAgent); err != nil {
		t.Errorf("Could not unmarshal response: %v", err)
	}

	if resAgent.ID != agentID {
		t.Errorf("Wrong agent ID. expect: %v, got: %v", agentID, resAgent.ID)
	}
}
