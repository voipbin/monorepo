package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_CampaignV1CampaigncallList(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  []cacampaigncall.Campaigncall
	}{
		{
			"normal",

			uuid.FromStringOrNil("61e0b6f6-6e2a-11ee-8da5-ef7ab5511ed0"),
			"2020-09-20 03:23:20.995000",
			10,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0dccc282-6e23-11ee-8173-23149728867a"}]`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/campaigncalls?page_token=%s&page_size=10", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			Data:     []byte(`{"customer_id":"61e0b6f6-6e2a-11ee-8da5-ef7ab5511ed0"}`),
			},
			[]cacampaigncall.Campaigncall{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0dccc282-6e23-11ee-8173-23149728867a"),
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

			filters := map[cacampaigncall.Field]any{
			cacampaigncall.FieldCustomerID: tt.customerID,
		}
		res, err := reqHandler.CampaignV1CampaigncallList(ctx, tt.pageToken, tt.pageSize, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_CampaignV1CampaigncallGetsByCampaignID(t *testing.T) {

	tests := []struct {
		name string

		campaignID uuid.UUID
		pageToken  string
		pageSize   uint64

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  []cacampaigncall.Campaigncall
	}{
		{
			"normal",

			uuid.FromStringOrNil("b2b0be5a-c859-11ec-acc0-c75b05c4cd00"),
			"2020-09-20 03:23:20.995000",
			10,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bd6d3594-c859-11ec-b2ed-af1657f376a7"}]`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/campaigncalls?page_token=%s&page_size=10", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"campaign_id":"b2b0be5a-c859-11ec-acc0-c75b05c4cd00"}`),
			},
			[]cacampaigncall.Campaigncall{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("bd6d3594-c859-11ec-b2ed-af1657f376a7"),
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

			filters := map[cacampaigncall.Field]any{
				cacampaigncall.FieldCampaignID: tt.campaignID,
			}
			res, err := reqHandler.CampaignV1CampaigncallList(ctx, tt.pageToken, tt.pageSize, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_CampaignV1CampaigncallGet(t *testing.T) {

	tests := []struct {
		name string

		campaigncallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaigncall.Campaigncall
	}{
		{
			"normal",

			uuid.FromStringOrNil("f3cff130-c859-11ec-ba02-4b142bed8c58"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f3cff130-c859-11ec-ba02-4b142bed8c58"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigncalls/f3cff130-c859-11ec-ba02-4b142bed8c58",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&cacampaigncall.Campaigncall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f3cff130-c859-11ec-ba02-4b142bed8c58"),
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

			res, err := reqHandler.CampaignV1CampaigncallGet(ctx, tt.campaigncallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaigncallDelete(t *testing.T) {

	tests := []struct {
		name string

		campaigncallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaigncall.Campaigncall
	}{
		{
			"normal",

			uuid.FromStringOrNil("08508d40-c85a-11ec-9a16-531149eb320b"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"08508d40-c85a-11ec-9a16-531149eb320b"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigncalls/08508d40-c85a-11ec-9a16-531149eb320b",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&cacampaigncall.Campaigncall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("08508d40-c85a-11ec-9a16-531149eb320b"),
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

			res, err := reqHandler.CampaignV1CampaigncallDelete(ctx, tt.campaigncallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
