package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_DomainCreate(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		DomainName      string
		DomainTmpName   string
		DomainTmpDetail string

		response  *rmdomain.Domain
		expectRes *rmdomain.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			"test.sip.voipbin.net",
			"test name",
			"test detail",

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("5b06161c-6ed9-11eb-85e4-f38ba2415baf"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
			&rmdomain.WebhookMessage{
				ID:         uuid.FromStringOrNil("5b06161c-6ed9-11eb-85e4-f38ba2415baf"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1DomainCreate(ctx, tt.agent.CustomerID, tt.DomainName, tt.DomainTmpName, tt.DomainTmpDetail).Return(tt.response, nil)

			res, err := h.DomainCreate(ctx, tt.agent, tt.DomainName, tt.DomainTmpName, tt.DomainTmpDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_DomainUpdate(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		id      uuid.UUID
		domainN string
		detail  string

		responseDomain *rmdomain.Domain
		expectRes      *rmdomain.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
			"update name",
			"update detail",

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
			&rmdomain.WebhookMessage{
				ID:         uuid.FromStringOrNil("d38cff42-6ed9-11eb-9117-5bf23c8e309c"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1DomainGet(ctx, tt.id).Return(tt.responseDomain, nil)
			mockReq.EXPECT().RegistrarV1DomainUpdate(ctx, tt.id, tt.domainN, tt.detail).Return(tt.responseDomain, nil)
			res, err := h.DomainUpdate(ctx, tt.agent, tt.id, tt.domainN, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_DomainDelete(t *testing.T) {

	type test struct {
		name     string
		agent    *amagent.Agent
		domainID uuid.UUID

		responseDomain *rmdomain.Domain
		expectRes      *rmdomain.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("4f7686fa-6eda-11eb-bc3f-5b6eefd85a3d"),

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("4f7686fa-6eda-11eb-bc3f-5b6eefd85a3d"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&rmdomain.WebhookMessage{
				ID:         uuid.FromStringOrNil("4f7686fa-6eda-11eb-bc3f-5b6eefd85a3d"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1DomainGet(ctx, tt.domainID).Return(tt.responseDomain, nil)
			mockReq.EXPECT().RegistrarV1DomainDelete(ctx, tt.domainID).Return(tt.responseDomain, nil)

			res, err := h.DomainDelete(ctx, tt.agent, tt.domainID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_DomainGet(t *testing.T) {

	type test struct {
		name     string
		agent    *amagent.Agent
		DomainID uuid.UUID

		response  *rmdomain.Domain
		expectRes *rmdomain.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),

			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
			&rmdomain.WebhookMessage{
				ID:         uuid.FromStringOrNil("8142024a-6eda-11eb-be4f-9b2b473fcf90"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1DomainGet(ctx, tt.DomainID).Return(tt.response, nil)

			res, err := h.DomainGet(ctx, tt.agent, tt.DomainID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_DomainGets(t *testing.T) {

	type test struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		response  []rmdomain.Domain
		expectRes []*rmdomain.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]rmdomain.Domain{
				{
					ID:         uuid.FromStringOrNil("cbd2f846-6eda-11eb-a1b5-c39b7ed749b1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					DomainName: "test.sip.voipbin.net",
					Name:       "test1",
					Detail:     "test detail1",
				},
				{
					ID:         uuid.FromStringOrNil("cf9ee9a8-6eda-11eb-8961-3b8e36c03336"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					DomainName: "test2.sip.voipbin.net",
					Name:       "test2",
					Detail:     "test detail2",
				},
			},
			[]*rmdomain.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("cbd2f846-6eda-11eb-a1b5-c39b7ed749b1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					DomainName: "test.sip.voipbin.net",
					Name:       "test1",
					Detail:     "test detail1",
				},
				{
					ID:         uuid.FromStringOrNil("cf9ee9a8-6eda-11eb-8961-3b8e36c03336"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					DomainName: "test2.sip.voipbin.net",
					Name:       "test2",
					Detail:     "test detail2",
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1DomainGets(ctx, tt.agent.CustomerID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.DomainGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
