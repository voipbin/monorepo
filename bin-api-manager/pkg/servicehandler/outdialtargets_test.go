package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_OutdialtargetCreate(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		outdialID         uuid.UUID
		outdialtargetName string
		detail            string
		data              string

		destination0 *commonaddress.Address
		destination1 *commonaddress.Address
		destination2 *commonaddress.Address
		destination3 *commonaddress.Address
		destination4 *commonaddress.Address

		responseOutdial *omoutdial.Outdial
		response        *omoutdialtarget.OutdialTarget
		expectRes       *omoutdialtarget.WebhookMessage
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

			uuid.FromStringOrNil("410fa394-8764-4300-a8b0-a6e6108c4208"),
			"test name",
			"test detail",
			"test data",

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000003",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000004",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000005",
			},

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("410fa394-8764-4300-a8b0-a6e6108c4208"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
			},
			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
			},
		},
		{
			"has 1 address",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("7aff596d-1db2-4456-95b4-bdab04296cd8"),
			"test name",
			"test detail",
			"test data",

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			nil,
			nil,
			nil,
			nil,

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("7aff596d-1db2-4456-95b4-bdab04296cd8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("895b86d9-4e58-4778-af61-d389dbeb9cf7"),
			},
			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("895b86d9-4e58-4778-af61-d389dbeb9cf7"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.responseOutdial, nil)
			mockReq.EXPECT().OutdialV1OutdialtargetCreate(ctx, tt.outdialID, tt.outdialtargetName, tt.detail, tt.data, tt.destination0, tt.destination1, tt.destination2, tt.destination3, tt.destination4).Return(tt.response, nil)
			res, err := h.OutdialtargetCreate(ctx, tt.agent, tt.outdialID, tt.outdialtargetName, tt.detail, tt.data, tt.destination0, tt.destination1, tt.destination2, tt.destination3, tt.destination4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialtargetGet(t *testing.T) {

	tests := []struct {
		name            string
		agent           *amagent.Agent
		outdialID       uuid.UUID
		outdialtargetID uuid.UUID

		responseOutdial *omoutdial.Outdial
		response        *omoutdialtarget.OutdialTarget
		expectRes       *omoutdialtarget.WebhookMessage
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
			uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
			uuid.FromStringOrNil("27092132-c523-11ec-8626-bb2b11494c8d"),

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&omoutdialtarget.OutdialTarget{
				ID:        uuid.FromStringOrNil("27092132-c523-11ec-8626-bb2b11494c8d"),
				OutdialID: uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
			},
			&omoutdialtarget.WebhookMessage{
				ID:        uuid.FromStringOrNil("27092132-c523-11ec-8626-bb2b11494c8d"),
				OutdialID: uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.responseOutdial, nil)
			mockReq.EXPECT().OutdialV1OutdialtargetGet(ctx, tt.outdialtargetID).Return(tt.response, nil)
			res, err := h.OutdialtargetGet(ctx, tt.agent, tt.outdialID, tt.outdialtargetID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialtargetGetsByOutdialID(t *testing.T) {

	type test struct {
		name      string
		agent     *amagent.Agent
		outdialID uuid.UUID
		pageToken string
		pageSize  uint64

		responseOutdial *omoutdial.Outdial
		response        []omoutdialtarget.OutdialTarget
		expectRes       []*omoutdialtarget.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("cf21cd20-c829-11ec-8452-6746e25a4103"),
			"2021-03-01 01:00:00.995000",
			10,

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("cf21cd20-c829-11ec-8452-6746e25a4103"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			[]omoutdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("cf484284-c829-11ec-8bde-6f873b300703"),
				},
			},
			[]*omoutdialtarget.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("cf484284-c829-11ec-8bde-6f873b300703"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.responseOutdial, nil)
			mockReq.EXPECT().OutdialV1OutdialtargetGetsByOutdialID(ctx, tt.outdialID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.OutdialtargetGetsByOutdialID(ctx, tt.agent, tt.outdialID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, tmp := range res {
				tmp.TMCreate = ""
				tmp.TMUpdate = ""
				tmp.TMDelete = ""
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_OutdialtargetDelete(t *testing.T) {

	tests := []struct {
		name            string
		agent           *amagent.Agent
		outdialID       uuid.UUID
		outdialtargetID uuid.UUID

		responseOutdial *omoutdial.Outdial
		response        *omoutdialtarget.OutdialTarget
		expectRes       *omoutdialtarget.WebhookMessage
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

			uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
			uuid.FromStringOrNil("f814109a-c62e-4cc3-8c8b-616fd91314a6"),

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&omoutdialtarget.OutdialTarget{
				ID:        uuid.FromStringOrNil("f814109a-c62e-4cc3-8c8b-616fd91314a6"),
				OutdialID: uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
			},
			&omoutdialtarget.WebhookMessage{
				ID:        uuid.FromStringOrNil("f814109a-c62e-4cc3-8c8b-616fd91314a6"),
				OutdialID: uuid.FromStringOrNil("1fc27dbe-2440-4e9d-b209-a8aa526e96d8"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.responseOutdial, nil)
			mockReq.EXPECT().OutdialV1OutdialtargetGet(ctx, tt.outdialtargetID).Return(tt.response, nil)
			mockReq.EXPECT().OutdialV1OutdialtargetDelete(ctx, tt.outdialtargetID).Return(tt.response, nil)
			res, err := h.OutdialtargetDelete(ctx, tt.agent, tt.outdialID, tt.outdialtargetID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
