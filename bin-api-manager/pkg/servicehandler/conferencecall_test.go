package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ConferencecallGet(t *testing.T) {

	tests := []struct {
		name             string
		agent            *amagent.Agent
		conferencecallID uuid.UUID

		responseConferencecall *cfconferencecall.Conferencecall

		expectRes *cfconferencecall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("54df63a2-15ad-11ed-9309-b3fd99910cf5"),

			&cfconferencecall.Conferencecall{
				ID:         uuid.FromStringOrNil("54df63a2-15ad-11ed-9309-b3fd99910cf5"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			&cfconferencecall.WebhookMessage{
				ID:         uuid.FromStringOrNil("54df63a2-15ad-11ed-9309-b3fd99910cf5"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferencecallGet(ctx, tt.conferencecallID).Return(tt.responseConferencecall, nil)

			res, err := h.ConferencecallGet(ctx, tt.agent, tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallGets(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		token string
		limit uint64

		response      []cfconferencecall.Conferencecall
		expectFilters map[string]string
		expectRes     []*cfconferencecall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			"2020-09-20T03:23:20.995000",
			10,

			[]cfconferencecall.Conferencecall{
				{
					ID: uuid.FromStringOrNil("7fc9cf36-50ca-11ee-9003-c76f7d344275"),
				},
			},
			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			[]*cfconferencecall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("7fc9cf36-50ca-11ee-9003-c76f7d344275"),
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

			mockReq.EXPECT().ConferenceV1ConferencecallGets(ctx, tt.token, tt.limit, tt.expectFilters).Return(tt.response, nil)
			res, err := h.ConferencecallGets(ctx, tt.agent, tt.limit, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallKick(t *testing.T) {

	tests := []struct {
		name             string
		agent            *amagent.Agent
		conferencecallID uuid.UUID

		responseConferencecall *cfconferencecall.Conferencecall

		expectRes *cfconferencecall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("c5291d7e-15ad-11ed-97e7-239ea4fba3e3"),

			&cfconferencecall.Conferencecall{
				ID:         uuid.FromStringOrNil("c5291d7e-15ad-11ed-97e7-239ea4fba3e3"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			&cfconferencecall.WebhookMessage{
				ID:         uuid.FromStringOrNil("c5291d7e-15ad-11ed-97e7-239ea4fba3e3"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferencecallGet(ctx, tt.conferencecallID).Return(tt.responseConferencecall, nil)
			mockReq.EXPECT().ConferenceV1ConferencecallKick(ctx, tt.conferencecallID).Return(tt.responseConferencecall, nil)

			res, err := h.ConferencecallKick(ctx, tt.agent, tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
