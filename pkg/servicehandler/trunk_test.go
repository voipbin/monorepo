package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	rmtrunk "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_TrunkCreate(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer

		trunkName  string
		detail     string
		domainName string
		authTypes  []rmtrunk.AuthType
		username   string
		password   string
		allowedIPs []string

		response  *rmtrunk.Trunk
		expectRes *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},

			trunkName:  "test name",
			detail:     "test detail",
			domainName: "test-domain",
			authTypes:  []rmtrunk.AuthType{rmtrunk.AuthTypeBasic, rmtrunk.AuthTypeIP},
			username:   "testusername",
			password:   "testpassword",
			allowedIPs: []string{"1.2.3.4"},

			response: &rmtrunk.Trunk{
				ID: uuid.FromStringOrNil("bc669058-54a5-11ee-999b-a3c9289707a5"),
			},

			expectRes: &rmtrunk.WebhookMessage{
				ID: uuid.FromStringOrNil("bc669058-54a5-11ee-999b-a3c9289707a5"),
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

			mockReq.EXPECT().RegistrarV1TrunkCreate(ctx, tt.customer.ID, tt.trunkName, tt.detail, tt.domainName, tt.authTypes, tt.username, tt.password, tt.allowedIPs).Return(tt.response, nil)

			res, err := h.TrunkCreate(ctx, tt.customer, tt.trunkName, tt.detail, tt.domainName, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkDelete(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer
		trunkID  uuid.UUID

		response  *rmtrunk.Trunk
		expectRes *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("02369970-54a6-11ee-8436-576a6c6712f2"),

			&rmtrunk.Trunk{
				ID:         uuid.FromStringOrNil("02369970-54a6-11ee-8436-576a6c6712f2"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&rmtrunk.WebhookMessage{
				ID:         uuid.FromStringOrNil("02369970-54a6-11ee-8436-576a6c6712f2"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().RegistrarV1TrunkGet(ctx, tt.trunkID).Return(tt.response, nil)
			mockReq.EXPECT().RegistrarV1TrunkDelete(ctx, tt.trunkID).Return(tt.response, nil)

			res, err := h.TrunkDelete(ctx, tt.customer, tt.trunkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkGet(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer
		trunkID  uuid.UUID

		response  *rmtrunk.Trunk
		expectRes *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("32309590-54a6-11ee-8466-9b257305288b"),

			&rmtrunk.Trunk{
				ID:         uuid.FromStringOrNil("32309590-54a6-11ee-8466-9b257305288b"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&rmtrunk.WebhookMessage{
				ID:         uuid.FromStringOrNil("32309590-54a6-11ee-8466-9b257305288b"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().RegistrarV1TrunkGet(ctx, tt.trunkID).Return(tt.response, nil)

			res, err := h.TrunkGet(ctx, tt.customer, tt.trunkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkGets(t *testing.T) {

	type test struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []rmtrunk.Trunk
		expectRes []*rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]rmtrunk.Trunk{
				{
					ID:         uuid.FromStringOrNil("6ac312f2-54a6-11ee-9e12-3bb0c25cd1e2"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				{
					ID:         uuid.FromStringOrNil("6af5fe56-54a6-11ee-8db0-b750622f6cc0"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
			},
			[]*rmtrunk.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("6ac312f2-54a6-11ee-9e12-3bb0c25cd1e2"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				{
					ID:         uuid.FromStringOrNil("6af5fe56-54a6-11ee-8db0-b750622f6cc0"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().RegistrarV1TrunkGetsByCustomerID(ctx, tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.TrunkGets(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkUpdateBasicInfo(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer

		id      uuid.UUID
		trunkName string
		detail  string
		authTypes []rmtrunk.AuthType
		username string
		password string
		allowedIPs []string

		response  *rmtrunk.Trunk
		expectRes *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},

			uuid.FromStringOrNil("bdf14732-54a6-11ee-b262-1f9dcbae10b2"),
			"update name",
			"update detail",
			[]rmtrunk.AuthType{rmtrunk.AuthTypeBasic},
			"updateusername",
			"updatepassword",
			[]string{"1.2.3.4"},

			&rmtrunk.Trunk{
				ID:         uuid.FromStringOrNil("bdf14732-54a6-11ee-b262-1f9dcbae10b2"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&rmtrunk.WebhookMessage{
				ID:         uuid.FromStringOrNil("bdf14732-54a6-11ee-b262-1f9dcbae10b2"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().RegistrarV1TrunkGet(ctx, tt.id).Return(&rmtrunk.Trunk{CustomerID: tt.customer.ID}, nil)
			mockReq.EXPECT().RegistrarV1TrunkUpdateBasicInfo(ctx, tt.id, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs).Return(tt.response, nil)
			res, err := h.TrunkUpdateBasicInfo(ctx, tt.customer, tt.id, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
