package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AuthLogin(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseAgent   *amagent.Agent
		responseCurTime string
	}{
		{
			name: "normal",

			username: "test@test.com",
			password: "testpassword",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6bc342d0-8aed-11ee-a07d-7bc7fee5a336"),
					CustomerID: uuid.FromStringOrNil("6c0ff198-8aed-11ee-8a04-474584947e03"),
				},
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1Login(ctx, gomock.Any(), tt.username, tt.password).Return(tt.responseAgent, nil)
			mockUtil.EXPECT().TimeGetCurTimeAdd(TokenExpiration).Return(tt.responseCurTime)

			res, err := h.AuthLogin(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == "" {
				t.Errorf("Expected non-empty token, got empty string")
			}

			// Parse the token and verify claims
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			claims, err := h.AuthJWTParse(ctx, res)
			if err != nil {
				t.Errorf("Could not parse token. err: %v", err)
			}

			// Verify "type" claim is "agent"
			tokenType, ok := claims["type"]
			if !ok {
				t.Errorf("Expected 'type' claim in token, but not found")
			}
			if tokenType != "agent" {
				t.Errorf("Wrong type claim. expected: agent, got: %v", tokenType)
			}

			// Verify "agent" claim exists
			if _, ok := claims["agent"]; !ok {
				t.Errorf("Expected 'agent' claim in token, but not found")
			}
		})
	}
}

func Test_AuthJWTGenerate(t *testing.T) {

	tests := []struct {
		name string

		data map[string]interface{}

		responseCurTime string

		expectRes map[string]interface{}
	}{
		{
			name: "normal",

			data: map[string]interface{}{
				"key1": "val1",
				"key2": "val2",
			},

			responseCurTime: "2023-11-19 09:29:11.763331118",
			expectRes: map[string]interface{}{
				"key1":   "val1",
				"key2":   "val2",
				"expire": "2023-11-19 09:29:11.763331118",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTimeAdd(TokenExpiration).Return(tt.responseCurTime)
			token, err := h.AuthJWTGenerate(tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			res, err := h.AuthJWTParse(ctx, token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
