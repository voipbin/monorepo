package listenhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
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


func Test_processV1AgentsPost_invalid_ring_method(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAgent := agenthandler.NewMockAgentHandler(mc)
	h := &listenHandler{
		agentHandler: mockAgent,
	}
	ctx := context.Background()

	mockAgent.EXPECT().Create(
		gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		agent.RingMethod("invalid"),
		gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(nil, cerrors.InvalidArgument(
		commonoutline.ServiceNameAgentManager,
		"INVALID_RING_METHOD",
		`unsupported ring_method "invalid": only "ringall" is supported`,
	))

	req := &sock.Request{
		URI: "/v1/agents",
		Data: []byte(`{
			"customer_id": "442f5d62-7f55-11ec-a2c0-0bcd3814d515",
			"username": "test@example.com",
			"password": "password",
			"ring_method": "invalid"
		}`),
	}

	res, err := h.processV1AgentsPost(ctx, req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
	}

	if res.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("Wrong data type. expect: %s, got: %s", cerrors.DataTypeVoipbinError, res.DataType)
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

func Test_processV1AgentsPost_nameLengthValidation(t *testing.T) {
	tests := []struct {
		name         string
		nameLen      int
		expectStatus int
	}{
		{"255 chars - ok", 255, 200},
		{"256 chars - too long", 256, 400},
		{"1000 chars - too long", 1000, 400},
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

			longName := strings.Repeat("A", tt.nameLen)
			reqBody, _ := json.Marshal(map[string]any{
				"customer_id": "92883d56-7fe3-11ec-8931-37d08180a2b9",
				"username":    "testuser@example.com",
				"password":    "TestPass123",
				"name":        longName,
				"detail":      "test",
				"ring_method": "ringall",
				"permission":  16,
				"tag_ids":     []string{},
				"addresses":   []string{},
			})

			if tt.expectStatus == 200 {
				mockAgent.EXPECT().Create(
					gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(),
					longName,
					gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(),
				).Return(&agent.Agent{}, nil)
			}

			req := &sock.Request{
				URI:  "/v1/agents",
				Data: reqBody,
			}

			res, err := h.processV1AgentsPost(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_processV1AgentsIDPut_nameLengthValidation(t *testing.T) {
	agentID := uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83")

	tests := []struct {
		name         string
		nameLen      int
		expectStatus int
	}{
		{"255 chars - ok", 255, 200},
		{"256 chars - too long", 256, 400},
		{"1000 chars - too long", 1000, 400},
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

			longName := strings.Repeat("A", tt.nameLen)
			reqBody, _ := json.Marshal(map[string]any{
				"name":        longName,
				"detail":      "test",
				"ring_method": "ringall",
			})

			if tt.expectStatus == 200 {
				mockAgent.EXPECT().UpdateBasicInfo(
					gomock.Any(),
					agentID,
					longName,
					gomock.Any(), gomock.Any(),
				).Return(&agent.Agent{}, nil)
			}

			req := &sock.Request{
				URI:  "/v1/agents/" + agentID.String(),
				Data: reqBody,
			}

			res, err := h.processV1AgentsIDPut(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}
