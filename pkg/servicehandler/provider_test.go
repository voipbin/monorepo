package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
)

func Test_ProviderGet(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer
		id       uuid.UUID

		response  *rmprovider.Provider
		expectRes *rmprovider.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},
			uuid.FromStringOrNil("0c6f3dd3-929e-4d3b-8231-5e8c10db6c21"),

			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("0c6f3dd3-929e-4d3b-8231-5e8c10db6c21"),
			},
			&rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("0c6f3dd3-929e-4d3b-8231-5e8c10db6c21"),
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

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.id).Return(tt.response, nil)

			res, err := h.ProviderGet(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ProviderGets(t *testing.T) {

	type test struct {
		name string

		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		responseProviders []rmprovider.Provider
		expectRes         []*rmprovider.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},
			"2021-03-01 01:00:00.995000",
			10,

			[]rmprovider.Provider{
				{
					ID: uuid.FromStringOrNil("d9603c2b-643c-43f2-9d58-71733785d45b"),
				},
			},
			[]*rmprovider.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("d9603c2b-643c-43f2-9d58-71733785d45b"),
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

			mockReq.EXPECT().RouteV1ProviderGets(ctx, tt.pageToken, tt.pageSize).Return(tt.responseProviders, nil)

			res, err := h.ProviderGets(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ProviderCreate(t *testing.T) {

	type test struct {
		name string

		customer     *cscustomer.Customer
		providerType rmprovider.Type
		hostname     string
		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		providerName string
		detail       string

		response  *rmprovider.Provider
		expectRes *rmprovider.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},
			rmprovider.TypeSIP,
			"test.com",
			"0001",
			"1000",
			map[string]string{
				"header_1": "val1",
				"header_2": "val2",
			},
			"test name",
			"test detail",

			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("c26e8f5b-5d5b-4618-a386-e633773f538e"),
			},
			&rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("c26e8f5b-5d5b-4618-a386-e633773f538e"),
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

			mockReq.EXPECT().RouteV1ProviderCreate(
				ctx,
				tt.providerType,
				tt.hostname,
				tt.techPrefix,
				tt.techPostfix,
				tt.techHeaders,
				tt.providerName,
				tt.detail,
			).Return(tt.response, nil)

			res, err := h.ProviderCreate(
				ctx,
				tt.customer,
				tt.providerType,
				tt.hostname,
				tt.techPrefix,
				tt.techPostfix,
				tt.techHeaders,
				tt.providerName,
				tt.detail,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ProviderDelete(t *testing.T) {

	type test struct {
		name string

		customer   *cscustomer.Customer
		providerID uuid.UUID

		responseProvider *rmprovider.Provider
		expectRes        *rmprovider.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},
			uuid.FromStringOrNil("3b889381-8944-49fa-8220-1a3b8b4d0894"),

			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("3b889381-8944-49fa-8220-1a3b8b4d0894"),
			},
			&rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("3b889381-8944-49fa-8220-1a3b8b4d0894"),
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

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.providerID).Return(tt.responseProvider, nil)
			mockReq.EXPECT().RouteV1ProviderDelete(ctx, tt.providerID).Return(tt.responseProvider, nil)

			res, err := h.ProviderDelete(ctx, tt.customer, tt.providerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ProviderUpdate(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer

		providerID   uuid.UUID
		providerType rmprovider.Type
		hostname     string
		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		providerName string
		detail       string

		responseProvider *rmprovider.Provider
		expectRes        *rmprovider.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},

			uuid.FromStringOrNil("9d4a55e6-f197-497a-a359-06d1858de39e"),
			rmprovider.TypeSIP,
			"test.com",
			"0001",
			"1000",
			map[string]string{
				"header_1": "val1",
				"header_2": "val2",
			},
			"update name",
			"update detail",

			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("9d4a55e6-f197-497a-a359-06d1858de39e"),
			},
			&rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("9d4a55e6-f197-497a-a359-06d1858de39e"),
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

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.providerID).Return(tt.responseProvider, nil)
			mockReq.EXPECT().RouteV1ProviderUpdate(
				ctx,
				tt.providerID,
				tt.providerType,
				tt.hostname,
				tt.techPrefix,
				tt.techPostfix,
				tt.techHeaders,
				tt.providerName,
				tt.detail,
			).Return(tt.responseProvider, nil)

			res, err := h.ProviderUpdate(
				ctx,
				tt.customer,
				tt.providerID,
				tt.providerType,
				tt.hostname,
				tt.techPrefix,
				tt.techPostfix,
				tt.techHeaders,
				tt.providerName,
				tt.detail,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
