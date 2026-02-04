package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	caoutplan "monorepo/bin-campaign-manager/models/outplan"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_CampaignV1OutplanCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		outplanName  string
		detail       string
		source       *address.Address
		dialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *caoutplan.Outplan
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("6a9320a2-c513-11ec-8d26-bfc178781416"),
			outplanName: "test name",
			detail:      "test detail",
			source: &address.Address{
				Type:   address.TypeTel,
				Target: "+821100000001",
			},
			dialTimeout:  30000,
			tryInterval:  600000,
			maxTryCount0: 5,
			maxTryCount1: 5,
			maxTryCount2: 5,
			maxTryCount3: 5,
			maxTryCount4: 5,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"99528bbc-c513-11ec-89b2-e3f0ee0792fc"}`),
			},

			expectTarget: "bin-manager.campaign-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/outplans",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"6a9320a2-c513-11ec-8d26-bfc178781416","name":"test name","detail":"test detail","source":{"type":"tel","target":"+821100000001"},"dial_timeout":30000,"try_interval":600000,"max_try_count_0":5,"max_try_count_1":5,"max_try_count_2":5,"max_try_count_3":5,"max_try_count_4":5}`),
			},
			expectResult: &caoutplan.Outplan{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("99528bbc-c513-11ec-89b2-e3f0ee0792fc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1OutplanCreate(
				ctx,
				tt.customerID,
				tt.outplanName,
				tt.detail,
				tt.source,
				tt.dialTimeout,
				tt.tryInterval,
				tt.maxTryCount0,
				tt.maxTryCount1,
				tt.maxTryCount2,
				tt.maxTryCount3,
				tt.maxTryCount4,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong matchdfdsfd.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1OutplanList(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  []caoutplan.Outplan
	}{
		{
			"normal",

			uuid.FromStringOrNil("4b1deb60-a784-4207-b1d8-a96df6bae951"),
			"2020-09-20T03:23:20.995000Z",
			10,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"b654bd9c-c514-11ec-962e-77d79eb3b3fe"}]`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/outplans?page_token=%s&page_size=10", url.QueryEscape("2020-09-20T03:23:20.995000Z")),
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			Data:     []byte(`{"customer_id":"4b1deb60-a784-4207-b1d8-a96df6bae951"}`),
			},
			[]caoutplan.Outplan{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("b654bd9c-c514-11ec-962e-77d79eb3b3fe"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			filters := map[caoutplan.Field]any{
				caoutplan.FieldCustomerID: tt.customerID,
			}
			res, err := reqHandler.CampaignV1OutplanList(ctx, tt.pageToken, tt.pageSize, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_CampaignV1OutplanGet(t *testing.T) {

	tests := []struct {
		name string

		outplanID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *caoutplan.Outplan
	}{
		{
			"normal",

			uuid.FromStringOrNil("0838f768-c515-11ec-a969-4fec734bbc81"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0838f768-c515-11ec-a969-4fec734bbc81"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/outplans/0838f768-c515-11ec-a969-4fec734bbc81",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&caoutplan.Outplan{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("0838f768-c515-11ec-a969-4fec734bbc81"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1OutplanGet(ctx, tt.outplanID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1OutplanDelete(t *testing.T) {

	tests := []struct {
		name string

		campaignID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *caoutplan.Outplan
	}{
		{
			"normal",

			uuid.FromStringOrNil("22d9075d-08bd-4eb0-b868-3b102f0bcb39"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"22d9075d-08bd-4eb0-b868-3b102f0bcb39"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/outplans/22d9075d-08bd-4eb0-b868-3b102f0bcb39",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&caoutplan.Outplan{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("22d9075d-08bd-4eb0-b868-3b102f0bcb39"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1OutplanDelete(ctx, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1OutplanUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		outplanID    uuid.UUID
		updateName   string
		updateDetail string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *caoutplan.Outplan
	}{
		{
			"normal",

			uuid.FromStringOrNil("63e29b96-c515-11ec-ba52-ab7d7001913f"),
			"update name",
			"update detail",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"63e29b96-c515-11ec-ba52-ab7d7001913f"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/outplans/63e29b96-c515-11ec-ba52-ab7d7001913f",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},
			&caoutplan.Outplan{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("63e29b96-c515-11ec-ba52-ab7d7001913f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1OutplanUpdateBasicInfo(ctx, tt.outplanID, tt.updateName, tt.updateDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1OutplanUpdateDialInfo(t *testing.T) {

	tests := []struct {
		name string

		outplanID    uuid.UUID
		source       *address.Address
		dialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *caoutplan.Outplan
	}{
		{
			name: "normal",

			outplanID: uuid.FromStringOrNil("e2b014d4-c516-11ec-a724-8bf87a1beb50"),
			source: &address.Address{
				Type:   address.TypeTel,
				Target: "+821100000001",
			},
			dialTimeout:  30000,
			tryInterval:  600000,
			maxTryCount0: 5,
			maxTryCount1: 5,
			maxTryCount2: 5,
			maxTryCount3: 5,
			maxTryCount4: 5,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e2b014d4-c516-11ec-a724-8bf87a1beb50"}`),
			},

			expectTarget: "bin-manager.campaign-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/outplans/e2b014d4-c516-11ec-a724-8bf87a1beb50/dials",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"source":{"type":"tel","target":"+821100000001"},"dial_timeout":30000,"try_interval":600000,"max_try_count_0":5,"max_try_count_1":5,"max_try_count_2":5,"max_try_count_3":5,"max_try_count_4":5}`),
			},
			expectResult: &caoutplan.Outplan{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e2b014d4-c516-11ec-a724-8bf87a1beb50"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1OutplanUpdateDialInfo(ctx, tt.outplanID, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
