package servicehandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/lib/common"
	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AuthLogin(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseAgent   *amagent.Agent
		responseCurTime string

		expectedRes string
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
			expectedRes:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhZ2VudCI6eyJpZCI6IjZiYzM0MmQwLThhZWQtMTFlZS1hMDdkLTdiYzdmZWU1YTMzNiIsImN1c3RvbWVyX2lkIjoiNmMwZmYxOTgtOGFlZC0xMWVlLThhMDQtNDc0NTg0OTQ3ZTAzIiwidXNlcm5hbWUiOiIiLCJwYXNzd29yZF9oYXNoIjoiIiwibmFtZSI6IiIsImRldGFpbCI6IiIsInJpbmdfbWV0aG9kIjoiIiwic3RhdHVzIjoiIiwicGVybWlzc2lvbiI6MCwidGFnX2lkcyI6bnVsbCwiYWRkcmVzc2VzIjpudWxsLCJ0bV9jcmVhdGUiOiIiLCJ0bV91cGRhdGUiOiIiLCJ0bV9kZWxldGUiOiIifSwiZXhwaXJlIjoiMjAyMy0xMS0xOSAwOToyOToxMS43NjMzMzExMTgifQ.E7PxZxY2R1T-nm-Rs5m-rAiDPZPmr-ySeNLmIKfQP_Y",
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
			mockUtil.EXPECT().TimeGetCurTimeAdd(common.TokenExpiration).Return(tt.responseCurTime)

			res, err := h.AuthLogin(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectedRes {
				t.Errorf("Wrong match. expected: %v, got: %v", res, tt.expectedRes)
			}
		})
	}
}
