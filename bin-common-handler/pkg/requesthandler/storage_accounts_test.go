package requesthandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	smaccount "monorepo/bin-storage-manager/models/account"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_StorageV1AccountCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *smaccount.Account
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("445c6bd8-1bc8-11ef-9397-5b14b39c0d70"),

			expectTarget: "bin-manager.storage-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/accounts",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"445c6bd8-1bc8-11ef-9397-5b14b39c0d70"}`),
			},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"44bd87a6-1bc8-11ef-b7ba-0f0f1468177a"}`),
			},
			expectResult: &smaccount.Account{
				ID: uuid.FromStringOrNil("44bd87a6-1bc8-11ef-b7ba-0f0f1468177a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1AccountCreate(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_StorageV1AccountGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *rabbitmqhandler.Response

		expectURL     string
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []smaccount.Account
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"44e914de-1bc8-11ef-ad71-0be6fe3f98ef"},{"id":"45176d8e-1bc8-11ef-9ab1-b3373adf14ce"}]`),
			},

			"/v1/accounts?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.storage-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/accounts?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			[]smaccount.Account{
				{
					ID: uuid.FromStringOrNil("44e914de-1bc8-11ef-ad71-0be6fe3f98ef"),
				},
				{
					ID: uuid.FromStringOrNil("45176d8e-1bc8-11ef-9ab1-b3373adf14ce"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1AccountGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_StorageV1AccountGet(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *smaccount.Account
	}{
		{
			"normal",

			uuid.FromStringOrNil("454865b0-1bc8-11ef-b131-932f42455765"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"454865b0-1bc8-11ef-b131-932f42455765"}`),
			},

			"bin-manager.storage-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/accounts/454865b0-1bc8-11ef-b131-932f42455765",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&smaccount.Account{
				ID: uuid.FromStringOrNil("454865b0-1bc8-11ef-b131-932f42455765"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1AccountGet(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_StorageV1AccountDelete(t *testing.T) {

	tests := []struct {
		name string

		accountID      uuid.UUID
		requestTimeout int

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *smaccount.Account
	}{
		{
			"normal",

			uuid.FromStringOrNil("bbef2ad2-1bc8-11ef-98ff-c36b990c2e2f"),
			5000,
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"bbef2ad2-1bc8-11ef-98ff-c36b990c2e2f"}`),
			},

			"bin-manager.storage-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/accounts/bbef2ad2-1bc8-11ef-98ff-c36b990c2e2f",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			&smaccount.Account{
				ID: uuid.FromStringOrNil("bbef2ad2-1bc8-11ef-98ff-c36b990c2e2f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1AccountDelete(ctx, tt.accountID, tt.requestTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
