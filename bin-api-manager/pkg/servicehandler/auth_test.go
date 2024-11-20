package servicehandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

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

		responseAgent *amagent.Agent
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1Login(ctx, gomock.Any(), tt.username, tt.password).Return(tt.responseAgent, nil)

			_, err := h.AuthLogin(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
